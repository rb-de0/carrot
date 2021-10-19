package main

import (
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

type TokenType int

const (
	TokenTypeHead TokenType = iota
	TokenTypeReserved
	TokenTypeIdentifier
	TokenTypeNumber
	TokenTypeEof
)

type Token struct {
	tokenType TokenType
	str       string
	value     int
	next      *Token
}

func Tokenize(input string) *Token {
	head := &Token{tokenType: TokenTypeHead, str: "head"}
	current := head
	i := 0
	runes := []rune(input)
	limit := len(runes)
	for i < limit {
		r := runes[i]
		isSpace := unicode.IsSpace(r)
		if isSpace {
			i++
			continue
		}
		c := string(r)
		prefixes := [...]string{"==", "!=", "<=", ">=", "var", "return", "if", "else", "fnc", "for", "break"}
		matchedPrefix := ""
		for _, prefix := range prefixes {
			prefixRunes := []rune(prefix)
			length := len(prefixRunes)
			if i+length < limit {
				searchRunes := runes[i : i+length]
				if reflect.DeepEqual(prefixRunes, searchRunes) {
					matchedPrefix = prefix
					i = i + length
					break
				}
			}
		}
		if matchedPrefix != "" {
			token := &Token{tokenType: TokenTypeReserved, str: matchedPrefix}
			current.next = token
			current = token
			continue
		}
		if unicode.IsLetter(r) {
			vi := i
			identifier := ""
			for {
				if vi >= limit {
					break
				}
				vc := runes[vi]
				if unicode.IsLetter(vc) {
					identifier = identifier + string(vc)
					vi++
				} else {
					break
				}
			}
			token := &Token{tokenType: TokenTypeIdentifier, str: identifier}
			current.next = token
			current = token
			i = vi
			continue
		}
		if strings.Contains("+-*/()=<>;{}", c) {
			token := &Token{tokenType: TokenTypeReserved, str: c}
			current.next = token
			current = token
			i++
			continue
		}
		value, err := strconv.Atoi(c)
		if err == nil {
			vi := i
			vc := ""
			resolvedValue := value
			for {
				if vi >= limit {
					break
				}
				vc = vc + string(runes[vi])
				value, err := strconv.Atoi(vc)
				if err == nil {
					resolvedValue = value
					vi++
				} else {
					break
				}
			}
			token := &Token{tokenType: TokenTypeNumber, value: resolvedValue, str: strconv.Itoa(resolvedValue)}
			current.next = token
			current = token
			i = vi
			continue
		}
		panic("Invalid Token")
	}
	eof := &Token{tokenType: TokenTypeEof, str: "eof"}
	current.next = eof
	return head.next
}
