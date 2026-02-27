package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
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

func (c Cipher) FindValidSums(maxSum int, wl WordList) []Triplet {
	result := []Triplet{}
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
				result = append(result, Triplet{
					iWord,
					validInfo[j],
					validInfo[i+j],
				})
			}
		}
	}

	return result
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

func WriteCSV(data []Triplet, filename string) {
	header := []string{"Numbers 1", "Letters 1", "Numbers 2", "Letters 2", "Numbers 3", "Letters 3"}
	table := [][]string{header}
	for _, t := range data {
		table = append(table, []string{t[0].numbers, t[0].letters, t[1].numbers, t[1].letters, t[2].numbers, t[2].letters})
	}

	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	w := csv.NewWriter(file)
	if err := w.WriteAll(table); err != nil {
		panic(err)
	}

	if err := w.Error(); err != nil {
		panic(err)
	}
}

func main() {
	wordList, err := MakeWordList("words.txt")
	if err != nil {
		fmt.Println("error loading word list:", err)
		os.Exit(1)
	}
	fmt.Println("word list loaded")
	cipher, err := CipherFromKey("wanderlust", false)
	if err != nil {
		fmt.Println("error creating cipher:", err)
		os.Exit(1)
	}
	data := cipher.FindValidSums(200000000, wordList)
	fmt.Println("writing CSV")
	WriteCSV(data, "wanderlust.csv")
	fmt.Println("done")
}
