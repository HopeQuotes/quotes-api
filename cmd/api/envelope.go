package main

type envelope map[string]interface{}

func getWrapper(data interface{}, message ...string) envelope {
	msg := "Success"
	if len(message) > 0 {
		msg = message[0]
	}
	return envelope{
		"message": msg,
		"data":    data,
	}
}
