package chain_test

import (
	"fmt"
	"os"
	"time"

	"github.com/augustoroman/sandwich/chain"
)

func ExampleFunc() {
	example := chain.Func{}.
		// Indicate that the chain will will receive a time.Duration as the first
		// arg when it's executed.
		Arg(time.Duration(0)).
		// When the chain is executed, it will first call time.Now which takes no
		// arguments but will return a time.Time value that will be available to
		// later calls.
		Then(time.Now).
		// Next, time.Sleep will be invoked, which requires a time.Duration
		// parameter. That's available since it's provided as an input to the chain.
		Then(time.Sleep).
		// Next, time.Since will be invoked, which requires a time.Time value that
		// was provided by the earlier time.Now call. It will return a time.Duration
		// value that will overwrite the input the chain.
		Then(time.Since).
		// Finally, we'll print out the stored time.Duration value.
		Then(func(dt time.Duration) {
			// Round to the nearest 10ms -- this makes the test not-flaky since the
			// sleep duration will not have been exact.
			dt = dt.Truncate(10 * time.Millisecond)
			fmt.Println("elapsed:", dt)
		})

	panicOnErr(example.Run(time.Duration(30 * time.Millisecond)))

	// Print the equivalent code:
	fmt.Println("Generated code is:")
	example.Code("example", "main", os.Stdout)

	// Output:
	// elapsed: 30ms
	// Generated code is:
	// func example(
	// ) func(
	// 	duration time.Duration,
	// ) {
	// 	return func(
	// 		duration time.Duration,
	// 	) {
	// 		var time_Time time.Time
	// 		time_Time = time.Now()
	//
	// 		time.Sleep(duration)
	//
	// 		duration = time.Since(time_Time)
	//
	// 		chain_test.ExampleFunc.func1(duration)
	//
	// 	}
	// }
}

func ExampleFunc_file() {
	// Chains can be used to do file operations!

	writeToFile := chain.Func{}.
		Arg("").          // filename
		Arg([]byte(nil)). // data
		Then(os.Create).
		Then((*os.File).Write).
		Then((*os.File).Close)

	// This never fails -- any errors in creating the file or writing to it will
	// be handled by the default error handler that logs a message, but the `Run`
	// itself doesn't fail unless the args are incorrect.
	panicOnErr(writeToFile.Run("test.txt", []byte("the data")))

	content, err := os.ReadFile("test.txt")
	panicOnErr(err)
	fmt.Printf("test.txt: %s\n", content)

	panicOnErr(os.Remove("test.txt"))

	// Output:
	// test.txt: the data
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
