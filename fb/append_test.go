package fb

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAppendJSON(t *testing.T) {
	Convey("Given an empty slice and some input json", t, func() {
		slice := []int{}
		input1 := []byte(`[1,2,3]`)
		input2 := []byte(`[4,5,6]`)

		Convey("after parsing the first input it the slice should contain the first numbers", func() {
			n, err := appendJSON(input1, &slice)
			So(err, ShouldBeNil)
			So(slice, ShouldHaveLength, 3)
			So(slice, ShouldResemble, []int{1, 2, 3})
			So(n, ShouldEqual, 3)

			Convey("after parsing more input, the old data should not be overwritten", func() {
				n, err := appendJSON(input2, &slice)
				So(err, ShouldBeNil)
				So(slice, ShouldHaveLength, 6)
				So(slice, ShouldResemble, []int{1, 2, 3, 4, 5, 6})
				So(n, ShouldEqual, 3)
			})
		})

		Convey("when passing a non pointer, it should return an error", func() {
			n, err := appendJSON(input1, slice)
			So(err, ShouldNotBeNil)
			So(n, ShouldEqual, 0)
		})

		Convey("when passing a pointer to a non-slice type, it should return an error", func() {
			n, err := appendJSON(input1, &struct{}{})
			So(err, ShouldNotBeNil)
			So(n, ShouldEqual, 0)
		})
	})
}
