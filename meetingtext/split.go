// Package meetingtext разбивает поле description так же, как фронт: до первого двойного
// перевода строки — название встречи, дальше — описание.
package meetingtext

import "strings"

const HeadingSep = "\n\n"

// SplitHeading возвращает заголовок и остаток текста после первого HeadingSep.
func SplitHeading(description string) (heading string, extra string) {
	d := strings.TrimSpace(description)
	if d == "" {
		return "", ""
	}
	i := strings.Index(d, HeadingSep)
	if i < 0 {
		return d, ""
	}
	return strings.TrimSpace(d[:i]), strings.TrimSpace(d[i+len(HeadingSep):])
}
