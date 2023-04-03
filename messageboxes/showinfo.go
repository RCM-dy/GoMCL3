package messageboxes

func ShowInfo(title, msg string) (err error) {
	_, err = MessageBox(title, msg, yesonly|informationICON)
	return
}
