package main

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// return later time
func GetLaterTimeStr(a, b string) (result string, err error) {
	timeA, err := http.ParseTime(a)
	if nil != err {
		log.Printf("[ERROR] failed to parse string %s as http time", a)
		return
	}
	timeB, err := http.ParseTime(b)
	if nil != err {
		log.Printf("[ERROR] failed to parse string %s as http time", b)
		return
	}

	if timeA.After(timeB) {
		result = a
	} else {
		result = b
	}

	return
}

// normalize url
func NormalizeURLStr(rawString string) string {
	if !strings.HasPrefix(rawString, HTTP_SCHEME) && !strings.HasPrefix(rawString, HTTPS_SCHEME) {
		rawString = HTTP_SCHEME + rawString
	}
	return rawString
}

// FeedTarget should be generated by ParseJsonConfig function
// find index regexp
func FindIndexRegs(feedTar *FeedTarget, feedURL *url.URL) []*regexp.Regexp {
	if 1 == len(feedTar.URLs) || 1 == len(feedTar.IndexRegs) {
		return feedTar.IndexRegs
	}
	for i := 0; i < len(feedTar.URLs); i++ {
		if feedTar.URLs[i] == feedURL {
			return []*regexp.Regexp{feedTar.IndexRegs[i]}
		}
	}
	return nil
}

// FeedTarget should be generated by ParseJsonConfig function
// find content regexp
func FindContentReg(feedTar *FeedTarget, feedURL *url.URL, indexReg *regexp.Regexp) *regexp.Regexp {
	if nil == indexReg {
		return nil
	}

	urlNum := len(feedTar.URLs)
	indNum := len(feedTar.IndexRegs)

	if 1 == len(feedTar.ContentRegs) || (1 == urlNum && 1 == indNum) {
		return feedTar.ContentRegs[0]
	}

	if 1 == indNum && 1 != urlNum {
		for i := 0; i < urlNum; i++ {
			if feedTar.URLs[i] == feedURL {
				return feedTar.ContentRegs[i]
			}
		}
	} else {
		for i := 0; i < indNum; i++ {
			if feedTar.IndexRegs[i] == indexReg {
				return feedTar.ContentRegs[i]
			}
		}
	}

	return nil
}

func FindIndexFilterReg(feedTar *FeedTarget, indexReg *regexp.Regexp) *regexp.Regexp {
	if 0 == len(feedTar.IndexFilterRegs) {
		return nil
	}

	if 1 == len(feedTar.IndexFilterRegs) {
		return feedTar.IndexFilterRegs[0]
	}

	for i := 0; i < len(feedTar.IndexRegs); i++ {
		if feedTar.IndexRegs[i] == indexReg {
			return feedTar.IndexFilterRegs[i]
		}
	}

	return nil
}

func FindContentFilterReg(feedTar *FeedTarget, contReg *regexp.Regexp) *regexp.Regexp {
	if 0 == len(feedTar.ContentFilterRegs) {
		return nil
	}

	if 1 == len(feedTar.ContentFilterRegs) {
		return feedTar.ContentFilterRegs[0]
	}

	for i := 0; i < len(feedTar.ContentRegs); i++ {
		if feedTar.ContentRegs[i] == contReg {
			return feedTar.ContentFilterRegs[i]
		}
	}

	return nil
}

// parse http Cache-Control response header
func ExtractMaxAge(cacheCtl string) (maxAge time.Duration) {
	for _, str := range strings.Split(cacheCtl, ",") {
		if strings.HasPrefix(str, "max-age") {
			maxAgeStrs := strings.Split(str, "=")
			if 2 != len(maxAgeStrs) {
				log.Printf("[ERROR] failed to parse max-age %s", str)
				return
			}
			maxAgeInt, err := strconv.Atoi(strings.TrimSpace(maxAgeStrs[1]))
			if nil != err {
				log.Printf("failed to convert max age string to int, originally %s, trimmed as %s: %s",
					maxAgeStrs[1], strings.TrimSpace(maxAgeStrs[1]), err)
			}
			return time.Duration(maxAgeInt)
		}
	}

	return
}

func FindPubDateReg(feedTar *FeedTarget, feedURL *url.URL) *regexp.Regexp {
	pubDateNum := len(feedTar.PubDateRegs)
	if 0 == pubDateNum {
		return nil
	} else if 1 == pubDateNum {
		return feedTar.PubDateRegs[0]
	}
	for i := 0; i < len(feedTar.URLs); i++ {
		if feedTar.URLs[i] == feedURL {
			return feedTar.PubDateRegs[i]
		}
	}
	return nil
}

func ParseDateMonth(monthStr string) (int, error) {
	var err error
	var month int

	// try integer month number
	month, err = strconv.Atoi(monthStr)
	if nil == err {
		return month, nil
	}

	// try short English month name
	monthDate, err := time.Parse("Jan", monthStr)
	if nil == err {
		return int(monthDate.Month()), nil
	}

	// try long English month name
	monthDate, err = time.Parse("January", monthStr)
	if nil == err {
		return int(monthDate.Month()), nil
	}

	return 0, errors.New("Failed to parse month string")
}

func ParsePubDate(formatReg *regexp.Regexp, dateStr string) (time.Time, error) {
	if nil == formatReg {
		log.Printf("[ERROR] error parsing pubdate, date format regexp is nil")
		return time.Time{}, errors.New("date format regexp is nil")
	}

	pubdateStr := TrimAllSpace(dateStr)
	if "" == pubdateStr {
		log.Printf("[ERROR] error parsing pubdate, pubdate string is empty")
		return time.Time{}, errors.New("pubdate string is empty")
	}

	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()
	hour := now.Hour()
	minute := now.Minute()
	second := now.Second()

	match := formatReg.FindSubmatch([]byte(dateStr))
	if nil == match {
		log.Printf("[ERROR] error parsing pubdate %s, pattern %s match failed", pubdateStr, formatReg.String())
		return time.Time{}, errors.New("failed to match pubdate pattern")
	}
	for patInd, patName := range formatReg.SubexpNames() {
		var err error
		patVal := string(match[patInd])
		switch patName {
		case PATTERN_YEAR:
			year, err = strconv.Atoi(patVal)
		case PATTERN_MONTH:
			month, err = ParseDateMonth(patVal)
		case PATTERN_DAY:
			day, err = strconv.Atoi(patVal)
		case PATTERN_HOUR:
			hour, err = strconv.Atoi(patVal)
		case PATTERN_MINUTE:
			minute, err = strconv.Atoi(patVal)
		case PATTERN_SECOND:
			second, err = strconv.Atoi(patVal)
		}

		if nil != err {
			log.Printf("[ERROR] error parsing pubdate: %s, time value %s cannot be parsed: %s",
				pubdateStr,
				match[patInd],
				err,
			)
			return time.Time{}, err
		}
	}

	return time.Date(year, time.Month(month), day, hour, minute, second, 0, GOFEED_DEFAULT_TIMEZONE), nil
}

func RemoveDuplicatEntries(feed *Feed) bool {
	if nil == feed {
		return false
	}

	entryMap := make(map[string]bool)
	newEntries := make([]*FeedEntry, 1)
	newEntryInd := 0

	for _, entry := range feed.Entries {
		if nil == entry.Link {
			continue
		}
		link := entry.Link.String()
		if !entryMap[link] {
			entryMap[link] = true
			newEntries = append(newEntries[:newEntryInd], entry)
			newEntryInd += 1
		} else {
			log.Printf("[WARN] removed duplicate feed entry %s", entry.Link.String())
		}
	}

	feed.Entries = newEntries

	return true
}

func SetPubDates(feed *Feed) {
	for _, entry := range feed.Entries {
		if nil == entry {
			log.Printf("[ERROR] failed to set pubDate: entry is nil")
			return
		}
		if nil != entry.PubDate {
			continue
		} else if nil != entry.Cache.LastModified {
			entry.PubDate = entry.Cache.LastModified
		} else if nil != entry.Cache.Date {
			entry.PubDate = entry.Cache.Date
		} else {
			log.Printf("[ERROR] entry's cache date is nil %s", entry.Link.String())
			now := time.Now()
			entry.PubDate = &now
		}
	}
}

func MinifyHtml(htmlData []byte) []byte {
	htmlData = HTML_WHITESPACE_REGEX.ReplaceAll(htmlData, HTML_WHITESPACE_REPL)
	htmlData = HTML_WHITESPACE_REGEX2.ReplaceAll(htmlData, HTML_WHITESPACE_REPL2)
	return htmlData
}

// remove the following tags in entry content
// <script>
func RemoveJunkContent(content []byte) []byte {
	if nil != content && len(content) > 0 {
		return HTML_SCRIPT_TAG.ReplaceAll(content, []byte(""))
	}
	return []byte("")
}

// generate pre-defined pattern name, PDP is short for Pre-defined Pattern
func GenPDPName(pdp string) string {
	return "{" + pdp + "}"
}

// generate pre-defined pattern regex string, PDP is short for Pre-defined Pattern
func GenPDPRegexStr(pdp string, nonEmpty bool, nonGreedy bool) string {
	regStr := `(?P<` + pdp + `>(?s).`

	if nonEmpty {
		regStr += `+`
	} else {
		regStr += `*`
	}

	if nonGreedy {
		regStr += `?`
	}

	return regStr + `)`
}

// trim all spaces including normal white-spaces, and full-length space(U+3000)
func TrimAllSpaces(source string) string {
	return strings.TrimFunc(source, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		switch r {
		case '\u3000':
			return true
		}
		return false
	})
}
