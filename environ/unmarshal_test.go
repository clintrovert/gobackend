package environ_test

import (
   "fmt"
   "math"
   "strconv"
   "testing"

   "github.com/clintrovert/gobackend/environ"
   "github.com/stretchr/testify/assert"
)

func TestUnmarshal_AllSupportedTypes_ShouldSucceed(t *testing.T) {
   type EnvironTest struct {
      TestString    string  `env:"TEST_STRING"`
      TestInt       int     `env:"TEST_INT"`
      TestInt32     int32   `env:"TEST_INT_32"`
      TestInt64     int64   `env:"TEST_INT_64"`
      TestBool      bool    `env:"TEST_BOOL"`
      TestFloat32   float32 `env:"TEST_FLOAT_32"`
      TestFloat64   float64 `env:"TEST_FLOAT_64"`
      TestNoNeedInt int
   }

   t.Setenv("TEST_STRING", "randString")
   t.Setenv("TEST_INT", "123456")
   t.Setenv("TEST_BOOL", "true")
   t.Setenv("TEST_INT_32", strconv.FormatInt(math.MaxInt32, 10))
   t.Setenv("TEST_INT_64", strconv.FormatInt(math.MaxInt64, 10))
   t.Setenv("TEST_FLOAT_32", fmt.Sprintf("%f", math.MaxFloat32))
   t.Setenv("TEST_FLOAT_64", fmt.Sprintf("%f", math.MaxFloat64))

   env := EnvironTest{}

   err := environ.Unmarshal(&env)
   assert.NoError(t, err)
   assert.Equal(t, "randString", env.TestString)
   assert.Equal(t, 123456, env.TestInt)
   assert.Equal(t, true, env.TestBool)
   assert.Equal(t, int32(math.MaxInt32), env.TestInt32)
   assert.Equal(t, int64(math.MaxInt64), env.TestInt64)
   assert.Equal(t, float32(math.MaxFloat32), env.TestFloat32)
   assert.Equal(t, float64(math.MaxFloat64), env.TestFloat64)
}
