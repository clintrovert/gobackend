package authn

import (
   "context"
   "errors"
   "log/slog"
   "strings"

   "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
   "google.golang.org/api/idtoken"
   "google.golang.org/grpc"
   "google.golang.org/grpc/codes"
   "google.golang.org/grpc/status"
)

// ClaimsContextKey is the context key for the JSON-structured list of claims
// present in a validated token.
const ClaimsContextKey = "jwt_claims"

var (
   // ErrProjectIdMissing indicates that the GCP Project for the Validator was
   // not provided.
   ErrProjectIdMissing = errors.New(
      "authn.GcpIdentifyProvider, GCP Project ID missing",
   )

   // ErrExpectedAudMissing indicates that the expected Audience (`aud`) in
   // the token claims was not provided.
   ErrExpectedAudMissing = errors.New(
      "authn.GcpIdentifyPlatformAuthenticator, expected token Audience missing",
   )
)

// GcpIdentifyPlatformAuthenticatorConfig handles environment variable mapping
//  of configuration values for GcpIdentifyPlatformAuthenticator.
type GcpIdentifyPlatformAuthenticatorConfig struct {
   ExpectedAudience string `env:"GCP_TOKEN_EXPECTED_AUDIENCE"`
   GcpProjectId     string `env:"GCP_PROJECT_ID"`
}

// GcpIdentifyPlatformAuthenticator handles authentication of JWT bearer tokens
// provided by GCP's Identify Platform.
type GcpIdentifyPlatformAuthenticator struct {
   expectedAudience string
   expectedIssuer   string

   // Some routes may not require authentication.
   publicMethods map[string]bool
}

// NewGcpIdentityPlatformValidator creates a new instance of
// GcpIdentifyPlatformAuthenticator with all required fields populated.
func NewGcpIdentityPlatformValidator(
   conf GcpIdentifyPlatformAuthenticatorConfig,
   publicMethods map[string]bool,
) (*GcpIdentifyPlatformAuthenticator, error) {
   if strings.TrimSpace(conf.GcpProjectId) == "" {
      return nil, ErrProjectIdMissing
   }

   if strings.TrimSpace(conf.ExpectedAudience) == "" {
      return nil, ErrExpectedAudMissing
   }

   return &GcpIdentifyPlatformAuthenticator{
      expectedIssuer:   "https://securetoken.google.com/" + conf.GcpProjectId,
      expectedAudience: conf.ExpectedAudience,
      publicMethods:    publicMethods,
   }, nil
}

// Authenticate authenticates an incoming bearer token to GCP's Identify
// Platform. Function meets the contract for go-grpc-middleware's AuthFunc
// defined here: https://github.com/grpc-ecosystem/go-grpc-middleware/blob/
// main/interceptors/auth/auth.go#L24
func (v *GcpIdentifyPlatformAuthenticator) Authenticate(
   ctx context.Context,
) (context.Context, error) {
   method, ok := grpc.Method(ctx)
   if ok {
      if _, isPublic := v.publicMethods[method]; isPublic {
         slog.Debug("Skipping authentication for public method: " + method)
         return ctx, nil
      }
   }

   token, err := auth.AuthFromMD(ctx, "bearer")
   if err != nil {
      slog.Error(
         "authn.GcpIdentifyPlatformAuthenticator, failed to parse token",
         "error", err.Error(),
      )

      return nil, status.Error(
         codes.Unauthenticated, "Authorization token not provided",
      )
   }

   validator, err := idtoken.NewValidator(ctx)
   if err != nil {
      slog.Error(
         "authn.GcpIdentifyPlatformAuthenticator, failed to token validator",
         "error", err.Error(),
      )

      return nil, status.Error(codes.Internal, "Authentication service error")
   }

   payload, err := validator.Validate(ctx, token, v.expectedAudience)
   if err != nil {
      slog.Error(
         "authn.GcpIdentifyPlatformAuthenticator, token validation failed",
         "error", err.Error(),
      )

      return nil, status.Error(
         codes.Unauthenticated, "Invalid authentication token",
      )
   }

   if payload.Issuer != v.expectedIssuer {
      slog.Error(
         "authn.GcpIdentifyPlatformAuthenticator, invalid token issuer",
         "expected", v.expectedIssuer,
         "actual", payload.Issuer,
      )

      return nil, status.Error(codes.Unauthenticated, "Invalid token issuer")
   }

   slog.Debug("successfully authenticated",
      "subject", payload.Subject,
      "email", payload.Claims["email"],
   )

   return context.WithValue(ctx, ClaimsContextKey, payload.Claims), nil
}
