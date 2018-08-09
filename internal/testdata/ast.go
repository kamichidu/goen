// +build testdata

package testing

type Testing struct {
	AnkoSuki string

	AnkoSokosoko string `ignore:""`

	AnkoKirai string

	ankoIranai string
}

type TestingEmbedded struct {
	*Testing

	AnkoNanisore string
}
