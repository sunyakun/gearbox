package util

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func NewCrashCatcher(crashHandler []func(any)) func() {
	return func() {
		err := recover()
		if err != nil {
			for _, handler := range crashHandler {
				handler(err)
			}
		}
	}
}
