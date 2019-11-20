package main

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}
