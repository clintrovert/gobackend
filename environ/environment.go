package environ

import (
   "errors"
   "fmt"
   "os"
   "strings"
)

var (
   ErrInvalidEnvironment = errors.New("invalid environment")
   ErrEnvironmentMissing = errors.New("environment was not provided")
)

const environmentVarName = "ENVIRONMENT"

type Environment int

const (
   Unknown     Environment = iota
   Development             = 1
   Staging                 = 2
   Production              = 3
   Test                    = 4
)

// String returns a string representation of Environment.
func (e Environment) String() string {
   switch e {
   case Test:
      return "test"
   case Development:
      return "dev"
   case Staging:
      return "stg"
   case Production:
      return "prd"
   case Unknown:
      return "unknown"
   default:
      return ""
   }
}

// ParseEnvironment takes in a string representation of Environment and
// returns the Environment.
func ParseEnvironment(s string) (Environment, error) {
   s = strings.ToLower(s)
   switch s {
   case "test", "testing", "tst":
      return Test, nil
   case "dev", "development":
      return Development, nil
   case "stg", "stage", "staging":
      return Staging, nil
   case "prd", "prod", "production":
      return Production, nil
   default:
      return Unknown, ErrInvalidEnvironment
   }
}

// GetEnvironment retrieves the environment from an environment variable.
func GetEnvironment() (Environment, error) {
   e := os.Getenv(environmentVarName)
   if e == "" {
      return Unknown, ErrEnvironmentMissing
   }

   env, err := ParseEnvironment(e)
   if err != nil {
      return Unknown, fmt.Errorf("environment misconfigured: %w", err)
   }

   return env, nil
}
