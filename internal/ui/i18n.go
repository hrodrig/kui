package ui

type Shell struct {
	I18n
	Links  []LocaleLink
	JSI18n string
}

type I18n struct {
	Locale string
	Lang   string
	T      func(string) string
	Tfmt   func(string, map[string]string) string
}

type LocaleLink struct {
	Code   string
	Label  string
	URL    string
	Active bool
}
