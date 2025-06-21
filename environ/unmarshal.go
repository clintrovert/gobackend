package environ

import (
   "errors"
   "fmt"
   "os"
   "reflect"
   "strconv"
   "strings"
)

const msgInvalidValueFmt = "invalid value '%s' for type '%s'"

var (
   // ErrMissingEnvVariable indicates the expected environment variable was
   // not provided.
   ErrMissingEnvVariable = errors.New("environ, missing env variable")
   // ErrNotSupportedTypeFound indicates the type that was expected is not
   // currently supported.
   ErrNotSupportedTypeFound = errors.New("environ, env type not supported")
   // ErrMalformedTag indicates that the field tag is not corrected formatted.
   ErrMalformedTag = errors.New("environ, malformed tag")
)

// Unmarshal parses the supplied config for the `env` tags on its fields and
// applies the associated env variables to the field value.
//
// e.g. fieldOne string `env:"MY_FIELD"` will apply the environment variable
// MY_FIELD to the value of fieldOne.
func Unmarshal(config any) error {
   v := reflect.ValueOf(config).Elem()
   t := reflect.TypeOf(config).Elem()
   var errs []error

   for i := 0; i < v.NumField(); i++ {
      var envErr error
      var errMsg string

      fieldVal := v.Field(i)
      fieldType := t.Field(i)

      if !fieldVal.CanSet() {
         continue
      }

      tagEncoded, ok := fieldType.Tag.Lookup("env")
      if !ok {
         continue
      }

      tag, optional, err := parseTagValue(tagEncoded)
      if err != nil {
         errMsg = fmt.Sprintf("env struct tag '%s' malformed", tagEncoded)
         envErr = fmt.Errorf("%s; %w", errMsg, ErrMalformedTag)
         errs = append(errs, envErr)

         continue
      }

      val, ok := os.LookupEnv(tag)
      if !ok && !optional {
         errMsg = fmt.Sprintf("required '%s' missing", tag)
         envErr = fmt.Errorf("%s; %w", errMsg, ErrMissingEnvVariable)
         errs = append(errs, envErr)

         continue
      }

      if !ok {
         continue
      }

      switch fieldType.Type.Kind() {
      case reflect.Bool:
         boolVal, err := strconv.ParseBool(val)
         if err != nil {
            errMsg = fmt.Sprintf(msgInvalidValueFmt, val, fieldType.Type.Name())
            envErr = fmt.Errorf("%s; %w", errMsg, err)
            errs = append(errs, envErr)

            continue
         }

         fieldVal.SetBool(boolVal)
      case reflect.String:
         fieldVal.SetString(val)
      case reflect.Float32, reflect.Float64:
         floatVal, err := strconv.ParseFloat(val, fieldType.Type.Bits())
         if err != nil {
            errMsg = fmt.Sprintf(msgInvalidValueFmt, val, fieldType.Type.Name())
            envErr = fmt.Errorf("%s; %w", errMsg, err)
            errs = append(errs, envErr)

            continue
         }

         fieldVal.SetFloat(floatVal)
      case reflect.Int, reflect.Int32, reflect.Int64:
         intVal, err := strconv.ParseInt(val, 10, fieldType.Type.Bits())
         if err != nil {
            errMsg = fmt.Sprintf(msgInvalidValueFmt, val, fieldType.Type.Name())
            envErr = fmt.Errorf("%s; %w", errMsg, err)
            errs = append(errs, envErr)

            continue
         }

         fieldVal.SetInt(intVal)
      default:
         errMsg = fmt.Sprintf(
            "found type '%s' is not supported",
            fieldType.Type.Name(),
         )
         envErr = fmt.Errorf("%s; %w", errMsg, ErrNotSupportedTypeFound)
         errs = append(errs, envErr)
      }
   }

   if len(errs) > 0 {
      return errors.Join(errs...)
   }

   return nil
}

func parseTagValue(value string) (envVar string, optional bool, err error) {
   parts := strings.Split(value, ",")
   for _, part := range parts {
      //nolint:gocritic
      if strings.EqualFold(part, "optional") {
         optional = true
      } else if envVar == "" {
         envVar = part
      } else {
         err = ErrMalformedTag
      }
   }

   return
}
