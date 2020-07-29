package builder

import "testing"

func CheckValidationTest(t *testing.T)  {
	b := Builder{}

	if err := b.CheckValidation(); err != nil {
		t.Errorf(err.Error())
	}
}
