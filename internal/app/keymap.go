package app

type pane int

const (
	paneScope pane = iota
	paneSkills
	paneDetail
)

type tab int

const (
	tabSkill tab = iota
	tabReferences
	tabScripts
	tabAssets
)

type subfocus int

const (
	subfocusList subfocus = iota
	subfocusContent
)
