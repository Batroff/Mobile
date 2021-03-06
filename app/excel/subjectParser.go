package excel

import (
	"github.com/batroff/schedule-back/models"
	"regexp"
	"strconv"
	"strings"
)

var searchForNumbersRegexp = regexp.MustCompile("(кр[ ])?((((\\d{2}|\\d)( *[ \\-,] *)){1,17}(\\d{2}|\\d))|(\\d{2}|\\d))") // регулярка для обнаружения чисел в строках
var searchForNumbersInBracesRegexp = regexp.MustCompile("\\((((\\d{2}|\\d)( *[ \\-,]? *)){1,17})+((нед\\.\\))|(нед\\)))") // регулярка для недель через тире запятую или пробел в скобках
var searchForNumbersWithDashNoBraces = regexp.MustCompile("((\\d{2}|\\d)-(\\d{2}|\\d)) нед\\.?")                          // регулярка для недель через тире без скобок
// var searchForNumbersWithDashNoBracesNoDot = regexp.MustCompile("(\\d{2}|\\d)-(\\d{2}|\\d) нед") // searchForNumbersWithDashNoBraces б эта строка была в NumbersIndex
var searchForNumbersRegexpCrutch1 = regexp.MustCompile("((кр\\.[ ]*)|(кр[ ]*))?(((\\d{2}|\\d)( *[ \\-,]? *)){1,17})+(((\\d{2}|\\d))|(\\d{2}|\\d)|([ ]*н\\.)|([ ]*н ))") // searchForNumbersRegexp для особых случаев
var searchForSlashAtEndOfLineInTeachers = regexp.MustCompile("[А-Яа-я]{1,100} [А-Я].[А-Я].\\/$")                                                                        // костыль на случай препод/\n препод/препод
var searchForNumbersWithDashDividedBySlash = regexp.MustCompile("\\((\\d{2}|\\d)-(\\d{2}|\\d) нед,\\/ (\\d{2}|\\d)-(\\d{2}|\\d) нед,\\)")                               // для поиска недель через - с запятой
var searchForNumbersWithDashDividedBySlashNoComma = regexp.MustCompile("\\((\\d{2}|\\d)-(\\d{2}|\\d) нед \\/ (\\d{2}|\\d)-(\\d{2}|\\d) нед\\)")                         // для поиска недель через - без запятой или точки
var searchForNumbersWithDashDividedBySlashWithDot = regexp.MustCompile("\\((\\d{2}|\\d)-(\\d{2}|\\d) нед\\.\\/ (\\d{2}|\\d)-(\\d{2}|\\d) нед\\.\\)")                    // для поиска недель через - с точкой
var searchForExcept = regexp.MustCompile("((^)|( ))кр((\\.)|(  ??))")                                                                                                   // ищет кр

func DefaultParse(subject, typeOfLesson, teacherName, cabinet, dayOfWeek, numberLesson, week string) []models.Lesson {
	RemoveJunk(&subject, &typeOfLesson)
	subject = strings.ReplaceAll(subject, "1С", "ОДИН ЦЕ")
	if subject != "" {
		//fmt.Println("исходная пара: ")
		//fmt.Println(subject)
		a, b, c, d := countBalance(SlashManage(SeparateLessons(subject), SeparateTeachers(teacherName), SeparateCabinets(cabinet), SeparateCabinets(typeOfLesson)))
		lessons := lessonBuilder(week, &a, &b, &c, &d)
		for i := range lessons {
			lessons[i].TeacherName = strings.ReplaceAll(lessons[i].TeacherName, "/", "")
			lessons[i].Subject = strings.ReplaceAll(lessons[i].Subject, "ОДИН ЦЕ", "1С")
			lessons[i].NumberLesson, _ = strconv.Atoi(numberLesson)
			lessons[i].DayOfWeek = dayOfWeek
			lessons[i].SubGroup = 0
			lessons[i].Exists = true
			//fmt.Println(lessons[i])
		}
		return lessons
	}

	return []models.Lesson{models.Lesson{
		Exists: false,
	}}
}
func lessonBuilder(week string, lessons, teachers, cabinets, types *[]string) []models.Lesson {
	var someLessons []models.Lesson
	for i, v := range *lessons {
		someLesson := models.NewLesson()
		flag := exceptFlag(v)
		FillingInOccurrenceLesson(flag, week, someLesson, v)
		someLesson.Subject = TruncateWeekNumbers(v)
		someLesson.TeacherName = (*teachers)[i]
		someLesson.Cabinet = (*cabinets)[i]
		someLesson.TypeOfLesson = (*types)[i]
		someLessons = append(someLessons, someLesson)
	}
	return someLessons
}
func numbersPresent(subject string) []int { // Возвращает номера недель в предмете subject

	var numbers []int
	subject = strings.ReplaceAll(subject, "кр", "")
	subject = strings.ReplaceAll(subject, ", ", ",")
	subject = strings.ReplaceAll(subject, " ,", ",")
	if DefaultRegexpNumbersIndex(subject)[0] > -1 {
		for _, v := range orSplit(subject[DefaultRegexpNumbersIndex(subject)[0]:DefaultRegexpNumbersIndex(subject)[1]]) {
			if strings.Contains(v, "-") {
				a, _ := strconv.Atoi(v[0:strings.Index(v, "-")])
				b, _ := strconv.Atoi(v[strings.Index(v, "-")+1:])
				for i := a; i < b; i++ {
					numbers = append(numbers, i)
				}
			} else {
				val, _ := strconv.Atoi(v)
				numbers = append(numbers, val)
			}
		}
	}
	return numbers
}
func orSplit(subject string) []string { // делит номера недель

	if strings.Contains(subject, ",") {
		return strings.Split(strings.ReplaceAll(subject, ".", ","), ",")
	} else {
		return strings.Split(subject, " ")
	}
}
func removeSpaces(subject string) string {
	for strings.Contains(subject, "  ") {
		subject = strings.ReplaceAll(subject, "  ", " ")
	}
	if subject[len(subject)-1:] == " " {
		subject = subject[:len(subject)-1]
	}
	return subject
}
func exceptFlag(subject string) bool {
	flag := false
	if searchForExcept.MatchString(subject) {
		flag = true
	}
	return flag
}
func NewLineSeparator(line string) []string {
	line = strings.ReplaceAll(line, "/", "\n")
	return strings.Split(line, "\n")
}
func RemoveJunk(subject, typeOfLesson *string) {
	//line = strings.ReplaceAll(line, ".", ",")
	*subject = strings.ReplaceAll(*subject, "\n", "")
	*typeOfLesson = strings.ReplaceAll(*typeOfLesson, "с/р", "c\\р")
}
func SeparateLessons(line string) []string {
	var lessons []string
	if searchForNumbersWithDashDividedBySlash.MatchString(line) {
		lessons = append(lessons, line[0:strings.Index(line, "(")-1]+line[strings.Index(line, "(")+1:strings.Index(line, "/")-1])
		lessons = append(lessons, line[0:strings.Index(line, "(")-1]+line[strings.Index(line, "/")+1:strings.Index(line, ")")-1])
	} else if searchForNumbersWithDashDividedBySlashNoComma.MatchString(line) {
		lessons = append(lessons, line[0:strings.Index(line, "(")-1]+line[strings.Index(line, "(")+1:strings.Index(line, "/")-1])
		lessons = append(lessons, line[0:strings.Index(line, "(")-1]+line[strings.Index(line, "/")+1:strings.Index(line, ")")])
	} else if searchForNumbersWithDashDividedBySlashWithDot.MatchString(line) {
		lessons = append(lessons, line[0:strings.Index(line, "(")-1]+line[strings.Index(line, "(")+1:strings.Index(line, "/")+1])
		lessons = append(lessons, line[0:strings.Index(line, "(")-1]+line[strings.Index(line, "/")+1:strings.Index(line, ")")])
	} else {
		for HasNextNumbers(line) > 0 {
			lessons = append(lessons, line[0:HasNextNumbers(line)])
			line = line[HasNextNumbers(line):]
		}
		if line != "" {
			lessons = append(lessons, line)
		}
	}
	SlashFix(&lessons)
	return lessons
}
func DefaultRegexpNumbersIndex(line string) []int { //Возвращает начальный и конечный индексы вхождения номеров недель в строку для строк вида 1, 2, 3, ..., 16 н
	if searchForNumbersRegexp.FindStringIndex(line) != nil { // Если в строке есть числа возвращает числа иначе -1000
		return searchForNumbersRegexp.FindStringIndex(line)
	} else {
		return []int{-1000, -1000} // вернуть -1000
	}
}
func NumbersIndex(line string) []int { //Возвращает начальный и конечный индексы вхождения номеров недель в строку отличную от стандартной
	if searchForNumbersInBracesRegexp.FindStringIndex(line) != nil {
		return searchForNumbersInBracesRegexp.FindStringIndex(line)
	}
	if searchForNumbersWithDashNoBraces.FindStringIndex(line) != nil {
		return searchForNumbersWithDashNoBraces.FindStringIndex(line)
	}
	if searchForNumbersRegexpCrutch1.FindStringIndex(line) != nil {
		return searchForNumbersRegexpCrutch1.FindStringIndex(line)
	} else {
		return []int{-1000, -1000}
	}
}
func HasNextNumbers(line string) int {
	line = strings.ReplaceAll(line, ".", ",")
	count := len(line)
	if DefaultRegexpNumbersIndex(line)[0] != -1000 {
		line = line[DefaultRegexpNumbersIndex(line)[1]:len(line)]
		count -= len(line)
		return DefaultRegexpNumbersIndex(line)[0] + count
	} else {
		return -1
	}
}
func FillingInOccurrenceLesson(flag bool, week string, someLesson models.Lesson, line string) {
	if DefaultRegexpNumbersIndex(line)[1] < 0 {
		someLesson.FillInWeeks(week)
	} else {
		for _, v := range numbersPresent(line) {
			if flag {
				someLesson.FillInWeeks(week)
			} else {
				if flag && week == "II" && (v-1)%2 != 0 {
					someLesson.OccurrenceLesson[v-1] = false
				} else if flag && week == "I" && (v-1)%2 == 0 {
					someLesson.OccurrenceLesson[v-1] = false
				} else if !flag && week == "II" && (v-1)%2 != 0 {
					someLesson.OccurrenceLesson[v-1] = true
				} else if !flag && week == "I" && (v-1)%2 == 0 {
					someLesson.OccurrenceLesson[v-1] = true
				}
			}
		}
	}
}
func SeparateTeachers(line string) []string {
	var teachers []string
	for strings.Contains(line, "\n") {
		teachers = append(teachers, line[0:strings.Index(line, "\n")])
		line = line[strings.Index(line, "\n")+1 : len(line)]
	}
	teachers = append(teachers, line)
	return teachers
}
func SeparateCabinets(line string) []string {
	var cabinets []string
	//for regexp.MustCompile("([А-Я]\\.){2}").MatchString(line){
	//	teachers = append(teachers, line[0:regexp.MustCompile("([А-Я]\\.){2}").FindStringIndex(line)[1]])
	//	line = line[regexp.MustCompile("([А-Я]\\.){2}").FindStringIndex(line)[1]:len(line)]
	//}
	for strings.Contains(line, "\n") {
		cabinets = append(cabinets, line[0:strings.Index(line, "\n")])
		line = line[strings.Index(line, "\n")+1 : len(line)]
	}
	cabinets = append(cabinets, line)
	return cabinets
}
func SlashFix(lessons *[]string) {
	for i, v := range *lessons {
		(*lessons)[i] = removeSpaces(v)
	}
	for i := 0; i < len(*lessons); i++ {
		if (*lessons)[i][len((*lessons)[i])-1:] == "/" {
			(*lessons)[i] = (*lessons)[i] + (*lessons)[i+1]
			copy((*lessons)[i+1:], (*lessons)[i+2:])
			*lessons = (*lessons)[:len(*lessons)-1]
		}
	}
}
func SlashManage(lessons, teachers, cabinets, types []string) ([]string, []string, []string, []string) {
	teachersFlag, cabinetFlag, typeFlag := false, false, false
	f := func(lessons, teachers, cabinets, types *[]string) bool {
		for _, l := range *lessons {
			if strings.Contains(l, "/") {
				return true
			}
			for _, t := range *teachers {
				if strings.Contains(t, "/") {
					return true
				}
			}
			teachersFlag = true
			for _, c := range *cabinets {
				if strings.Contains(c, "/") {
					return true
				}
			}
			cabinetFlag = true
			for _, t := range *types {
				if strings.Contains(t, "/") {
					return true
				}
			}
			typeFlag = true
		}
		return false
	}
	for f(&lessons, &teachers, &cabinets, &types) {
		for j, l := range lessons {
			if strings.Contains(l, "/") {
				SliceSlashManage(j, &lessons)
			}
		}

		for i, t := range teachers {
			if !(len(lessons) > len(teachers)) {
				teachersFlag = true
				continue
			}
			if strings.Contains(t, "/") {
				SliceSlashManage(i, &teachers)
			}
		}
		for i, c := range cabinets {
			if !(len(lessons) > len(cabinets)) {
				cabinetFlag = true
				continue
			}
			if strings.Contains(c, "/") {
				SliceSlashManage(i, &cabinets)
			}
		}
		for i, t := range types {
			if !(len(lessons) > len(types)) {
				typeFlag = true
				continue
			}
			if strings.Contains(t, "/") {
				SliceSlashManage(i, &types)
			}
		}
		if typeFlag && cabinetFlag && teachersFlag {
			return lessons, teachers, cabinets, types
		}
	}
	return lessons, teachers, cabinets, types
}
func SliceSlashManage(i int, slice *[]string) { // Разбивает элементы слайса по слешам
	partBefore := ""
	if HasNextNumbers((*slice)[i]) < 0 && DefaultRegexpNumbersIndex((*slice)[i])[0] > 0 && !strings.Contains((*slice)[i], "-") {
		partBefore = (*slice)[i][0 : DefaultRegexpNumbersIndex((*slice)[i])[1]+4] // часть с кроме или номерами недель
		(*slice)[i] = (*slice)[i][DefaultRegexpNumbersIndex((*slice)[i])[1]+4:]
	}
	if searchForSlashAtEndOfLineInTeachers.MatchString((*slice)[i]) { // костыль на случай препод/\n препод/препод
		(*slice)[i] = partBefore + (*slice)[i][0:strings.Index((*slice)[i], "/")]
	} else if strings.Contains((*slice)[i], "/") {
		*slice = append(*slice, "")
		copy((*slice)[i+1:], (*slice)[i:])
		(*slice)[i] = partBefore + (*slice)[i][0:strings.Index((*slice)[i], "/")]
		(*slice)[i+1] = partBefore + (*slice)[i+1][strings.Index((*slice)[i+1], "/")+1:len((*slice)[i+1])]
	}
}
func countBalance(lessons, teachers, cabinets, types []string) ([]string, []string, []string, []string) { //
	if len(lessons) > len(teachers) {
		lessons, teachers = balanceSlices(lessons, teachers)
	}
	if len(lessons) > len(cabinets) {
		lessons, cabinets = balanceSlices(lessons, cabinets)
	}
	if len(lessons) > len(types) {
		lessons, types = balanceSlices(lessons, types)
	}
	return lessons, teachers, cabinets, types
}
func balanceSlices(lessons, teachers []string) ([]string, []string) { // балансит кол-во предметов учителей кабинетов типов предмета
	if len(lessons) == len(teachers)*2 {
		for len(lessons) != len(teachers) {
			teachers = append(teachers, "")
		}
		if len(lessons) == 4 {
			teachers[3] = teachers[1]
			teachers[2] = teachers[1]
			teachers[1] = teachers[0]
		} else if len(lessons) == 2 {
			teachers[1] = teachers[0]
		}
	} else if len(teachers) == 1 {
		for i := 1; i < len(lessons); i++ {
			teachers = append(teachers, teachers[0])
		}
	} else if len(teachers) == 2 && len(lessons) == 3 {
		teachers = append(teachers, teachers[0])
	} else if len(teachers) == 3 && len(lessons) == 4 {
		teachers = append(teachers, teachers[0])
	}
	return lessons, teachers
}
func TruncateWeekNumbers(lesson string) string {
	//if regexp.MustCompile(" *$").FindStringIndex(*lesson) != nil {
	//	{
	//		*lesson = (*lesson)[:regexp.MustCompile(" *$").FindStringIndex(*lesson)[0]]
	//	}

	if NumbersIndex(lesson)[1] > 0 {
		if len(lesson) == NumbersIndex(lesson)[1] {
			lesson = (lesson)[:NumbersIndex(lesson)[0]]
		} else {
			lesson = (lesson)[NumbersIndex(lesson)[1]:]
		}
	}
	return lesson
}
