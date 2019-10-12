package main

func defaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
