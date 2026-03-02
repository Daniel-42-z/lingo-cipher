package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

type Cipher struct {
	letterToNumber map[rune]rune
	numberToLetter map[rune]rune
	base           int
}

func CipherFromKey(k string, leading0 bool) (Cipher, error) {
	length := len(k)
	if length >= 36 {
		return Cipher{}, errors.New("key longer than 36 characters")
	}
	letters := []rune{}
	for _, l := range k {
		if slices.Contains(letters, l) {
			return Cipher{}, errors.New("key contains repeat letters")
		}
		letters = append(letters, l)
	}
	numbers, err := MakeNumbers(length, leading0)
	if err != nil {
		return Cipher{}, err
	}

	letterToNumber := make(map[rune]rune, length)
	numberToLetter := make(map[rune]rune, length)
	for i := range length {
		letterToNumber[letters[i]] = numbers[i]
		numberToLetter[numbers[i]] = letters[i]
	}
	return Cipher{letterToNumber, numberToLetter, length}, nil
}

func MakeNumbers(l int, leading0 bool) ([]rune, error) {
	if l >= 36 {
		return nil, errors.New("cipher key too long")
	}
	numbers := []rune{}
	lengthWithout0 := l - 1
	if lengthWithout0 <= 8 {
		for i := range lengthWithout0 {
			numbers = append(numbers, rune('0'+i+1))
		}
	} else {
		for i := range 9 {
			numbers = append(numbers, rune('0'+i+1))
		}
		lettersLength := lengthWithout0 - 9
		for i := range lettersLength {
			numbers = append(numbers, rune('a'+i))
		}
	}
	if leading0 {
		return append([]rune{rune('0')}, numbers...), nil
	}
	return append(numbers, rune('0')), nil
}

func (c Cipher) FromLetters(letters string) string {
	numbers := make([]rune, 0, len(letters))
	for _, l := range letters {
		numbers = append(numbers, c.letterToNumber[l])
	}
	return string(numbers)
}

func (c Cipher) FromNumbers(numbers string) string {
	letters := make([]rune, 0, len(numbers))
	for _, n := range numbers {
		letters = append(letters, c.numberToLetter[n])
	}
	return string(letters)
}

func BaseAdd(n1, n2 string, b int) (string, error) {
	val1, err := strconv.ParseInt(n1, b, 64)
	if err != nil {
		return "", fmt.Errorf("invalid base-11 string n1: %v", err)
	}
	val2, err := strconv.ParseInt(n2, b, 64)
	if err != nil {
		return "", fmt.Errorf("invalid base-11 string n2: %v", err)
	}
	sum := val1 + val2
	return strconv.FormatInt(sum, b), nil
}

func BaseTimes(n1, n2 string, b int) (string, error) {
	val1, err := strconv.ParseInt(n1, b, 64)
	if err != nil {
		return "", fmt.Errorf("invalid base-11 string n1: %v", err)
	}
	val2, err := strconv.ParseInt(n2, b, 64)
	if err != nil {
		return "", fmt.Errorf("invalid base-11 string n2: %v", err)
	}
	product := val1 * val2
	return strconv.FormatInt(product, b), nil
}

type WordList map[string]struct{}

func MakeWordList(fileName string) (WordList, error) {
	wordList := make(map[string]struct{})
	data, err := os.ReadFile(fileName)
	if err != nil {
		return wordList, err
	}

	for word := range strings.SplitSeq(string(data), "\n") {
		word = strings.TrimSpace(strings.ToLower(word))
		if word != "" {
			wordList[word] = struct{}{}
		}
	}
	return wordList, nil
}

func IsValidWord(w string, wl WordList) bool {
	_, ok := wl[w]
	return ok
}

type Word struct {
	numbers string
	letters string
}

type Triplet [3]Word

func (c Cipher) FindValidSums(maxSum int, wl WordList, action func(Triplet)) {
	maxNumber := maxSum / 2

	validInfo := make([]Word, maxSum)
	isValid := make([]bool, maxSum)
	validNumbers := make([]int, 0)

	for k := range maxSum {
		numbers, letters := c.fromInt(k)
		if IsValidWord(letters, wl) {
			validInfo[k] = Word{numbers, letters}
			isValid[k] = true
			validNumbers = append(validNumbers, k)
		}
	}

	for _, i := range validNumbers {
		if i >= maxNumber {
			break
		}
		iWord := validInfo[i]

		for _, j := range validNumbers {
			if j < i {
				continue
			}
			if j >= maxNumber {
				break
			}

			if isValid[i+j] {
				action(Triplet{
					iWord,
					validInfo[j],
					validInfo[i+j],
				})
			}
		}
	}
}

func (c Cipher) fromInt(val int) (string, string) {
	if c.base == 10 {
		numbers := strconv.Itoa(val)
		letters := c.FromNumbers(numbers)
		return numbers, letters
	}
	numbers := strconv.FormatInt(int64(val), c.base)
	letters := c.FromNumbers(numbers)
	return numbers, letters
}

func MakeCSVWriterAction(w *csv.Writer) func(Triplet) {
	return func(t Triplet) {
		record := []string{t[0].numbers, t[0].letters, t[1].numbers, t[1].letters, t[2].numbers, t[2].letters}
		if err := w.Write(record); err != nil {
			fmt.Println("error writing to csv:", err)
			os.Exit(1)
		}
	}
}

func main() {
	var (
		wordListPath string
		upperBound   int
		key          string
		leading0     bool
		outputPath   string
	)

	pflag.StringVarP(&wordListPath, "word-list", "w", "words.txt", "Path to word list used")
	pflag.IntVarP(&upperBound, "max", "m", 200000, "Max value of the sum (in base 10)")
	pflag.StringVarP(&key, "key", "k", "wanderlust", "cipher")
	pflag.BoolVarP(&leading0, "leading0", "0", false, "Whether to start the \"numbers\" list with 0")
	pflag.StringVarP(&outputPath, "output", "o", "", "File path to output CSV")
	pflag.Lookup("output").DefValue = "<key>-<max>[-0].csv"
	pflag.Parse()

	if !pflag.Lookup("output").Changed {
		suffix := ""
		if leading0 {
			suffix = "-0"
		}

		// Format: key-upperBound[-0].csv
		outputPath = key + "-" + strconv.Itoa(upperBound) + suffix + ".csv"
	}

	wordList, err := MakeWordList(wordListPath)
	if err != nil {
		fmt.Println("error loading word list:", err)
		os.Exit(1)
	}
	cipher, err := CipherFromKey(key, leading0)
	if err != nil {
		fmt.Println("error creating cipher:", err)
		os.Exit(1)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("error creating csv file:", err)
		os.Exit(1)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Println("error closing csv file:", err)
			os.Exit(1)
		}
	}()

	w := csv.NewWriter(file)
	defer w.Flush()

	header := []string{"Numbers 1", "Letters 1", "Numbers 2", "Letters 2", "Numbers 3", "Letters 3"}
	if err := w.Write(header); err != nil {
		fmt.Println("error writing csv header:", err)
		os.Exit(1)
	}

	cipher.FindValidSums(upperBound, wordList, MakeCSVWriterAction(w))

	if err := w.Error(); err != nil {
		fmt.Println("error flushing csv:", err)
		os.Exit(1)
	}
}
