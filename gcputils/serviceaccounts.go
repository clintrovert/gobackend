package main

import (
   "context"
   "fmt"
   "log/slog"

   iamadmin "cloud.google.com/go/iam/admin/apiv1"
   iamadminpb "cloud.google.com/go/iam/admin/apiv1/adminpb"
   iampolicy "cloud.google.com/go/iam/apiv1"

   "cloud.google.com/go/iam/apiv1/iampb"
)

// M2MServiceAccount holds the details for a newly created M2M client
type M2MServiceAccount struct {
   Email string `json:"client_id"`
   // PrivateKey is Base64 encoded private key data
   PrivateKey string `json:"private_key"`
   // KeyID is the generated key
   KeyID       string `json:"key_id"`
   DisplayName string `json:"display_name"`
   // ServiceAccountID is the unique ID.
   ServiceAccountID string `json:"service_account_id"`
}

// NewM2MServiceAccount creates a new GCP service account for M2M
// authentication and generates a key for it.
func NewM2MServiceAccount(
   ctx context.Context,
   projectID string,
   clientID string,
   displayName string,
) (*M2MServiceAccount, error) {
   iamAdminClient, err := iamadmin.NewIamClient(ctx)
   if err != nil {
      return nil, fmt.Errorf("iamadmin.NewIAMClient: %w", err)
   }
   defer iamAdminClient.Close()

   saParent := fmt.Sprintf("projects/%s", projectID)
   saRequest := &iamadminpb.CreateServiceAccountRequest{
      Name: saParent,
      ServiceAccount: &iamadminpb.ServiceAccount{
         DisplayName: displayName,
         Description: fmt.Sprintf("M2M client SA for %s", clientID),
      },
      AccountId: clientID,
   }

   slog.Info("Creating SA named %s...", clientID)
   createdSA, err := iamAdminClient.CreateServiceAccount(ctx, saRequest)
   if err != nil {
      return nil, fmt.Errorf("CreateServiceAccount: %w", err)
   }
   slog.Info("Service Account created",
      "email", createdSA.Email,
      "name", createdSA.Name,
   )

   // WARNING: private_key_data is returned ONLY ONCE.
   // Must be stored securely.
   keyRequest := &iamadminpb.CreateServiceAccountKeyRequest{
      Name:         createdSA.Name,
      KeyAlgorithm: iamadminpb.ServiceAccountKeyAlgorithm_KEY_ALG_RSA_2048,
      // nolint: lll
      PrivateKeyType: iamadminpb.ServiceAccountPrivateKeyType_TYPE_GOOGLE_CREDENTIALS_FILE,
   }

   slog.Info("Generating key for service account", "account", createdSA.Email)
   generatedKey, err := iamAdminClient.CreateServiceAccountKey(ctx, keyRequest)
   if err != nil {
      if delErr := iamAdminClient.DeleteServiceAccount(
         ctx, &iamadminpb.DeleteServiceAccountRequest{
            Name: createdSA.Name,
         },
      ); delErr != nil {
         slog.Error("Failed to clean up service account",
            "account", createdSA.Email, "error", delErr.Error(),
         )
      }

      return nil, fmt.Errorf("CreateServiceAccountKey: %w", err)
   }

   // .g. projects/project-id/serviceAccounts/email/keys/key-id
   slog.Info("Key created",
      "account", createdSA.Email, "key ID", generatedKey.Name,
   )

   return &M2MServiceAccount{
      Email:            createdSA.Email,
      PrivateKey:       string(generatedKey.PrivateKeyData),
      KeyID:            generatedKey.Name,
      DisplayName:      createdSA.DisplayName,
      ServiceAccountID: clientID,
   }, nil
}

// grantRolesToServiceAccount grants specific IAM roles to a service account
// at the project level.
func grantRolesToServiceAccount(
   ctx context.Context,
   projectID string,
   serviceAccountEmail string,
   roles []string,
) error {
   // Initialize the IAM Policy client
   iamPolicyClient, err := iampolicy.NewIamPolicyClient(ctx)
   if err != nil {
      return fmt.Errorf("iampolicy.NewIamPolicyClient: %w", err)
   }
   defer iamPolicyClient.Close()

   resource := fmt.Sprintf("projects/%s", projectID)

   getPolicyReq := &iampb.GetIamPolicyRequest{
      Resource: resource,
   }
   policy, err := iamPolicyClient.GetIamPolicy(ctx, getPolicyReq)
   if err != nil {
      return fmt.Errorf("GetIamPolicy: %w", err)
   }

   member := fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)
   for _, roleName := range roles {
      foundRole := false
      for _, binding := range policy.Bindings {
         if binding.Role == roleName {
            // Add member if not already present
            foundMember := false
            for _, m := range binding.Members {
               if m == member {
                  foundMember = true
                  break
               }
            }
            if !foundMember {
               binding.Members = append(binding.Members, member)
            }
            foundRole = true
            break
         }
      }
      if !foundRole {
         policy.Bindings = append(policy.Bindings, &iampb.Binding{
            Role:    roleName,
            Members: []string{member},
         })
      }
   }

   setPolicyReq := &iampb.SetIamPolicyRequest{
      Resource: resource,
      Policy:   policy,
   }
   _, err = iamPolicyClient.SetIamPolicy(ctx, setPolicyReq)
   if err != nil {
      return fmt.Errorf("SetIamPolicy: %w", err)
   }

   slog.Info("Granted roles to service account",
      "roles", roles, "account", serviceAccountEmail, "project", projectID,
   )

   return nil
}
