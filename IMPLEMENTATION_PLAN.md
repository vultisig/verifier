# Verifier Implementation Plan

This document provides a detailed roadmap for completing the migration of the Verifier service from the monolithic repository to a standalone service.

## Priority Tasks

### 1. TSS Integration (High Priority)

The most critical component is the integration with the TSS library for cryptographic operations:

- [ ] Integrate DKLS library for threshold signature scheme operations
- [ ] Implement key generation flow with proper error handling
- [ ] Create a complete signing flow with DKLS
- [ ] Add signature verification
- [ ] Update sigutil package to use actual TSS implementation

Implementation notes:

- The current code in `internal/sigutil/signing.go` contains placeholders
- Need to implement proper integration with the TSS library from `github.com/vultisig/mobile-tss-lib/tss`
- Focus only on DKLS support as GG20 is being deprecated

### 2. API Endpoint Completion (High Priority)

Complete the implementation of all API endpoints:

- [ ] `SignMessages`: Complete implementation with actual TSS signing
- [ ] `ReshareVault`: Implement the vault resharing logic
- [ ] `GetVault`: Implement vault retrieval with proper ACLs
- [ ] `ExistVault`: Complete the vault existence check
- [ ] `GetKeysignResult`: Implement result retrieval
- [ ] Policy endpoints: Complete CRUD operations
- [ ] Transaction endpoints: Complete transaction handling

Implementation notes:

- Most endpoints have stubs in `internal/api/server.go`
- Use the existing database structure in `internal/storage/postgres/postgres.go`
- Implement proper error handling with appropriate HTTP status codes

### 3. Authentication and Security (Medium Priority)

Improve the authentication and security mechanisms:

- [ ] Enhance the AuthService implementation in `internal/service/auth.go`
- [ ] Add proper JWT validation and expiration handling
- [ ] Implement role-based access control for APIs
- [ ] Add secure password handling for admin users
- [ ] Implement proper input validation on all endpoints

Implementation notes:

- Currently using a basic JWT implementation
- Need to add proper token refresh, expiration, and revocation
- Consider adding rate limiting for sensitive endpoints

### 4. Worker and Task Processing (Medium Priority)

Complete the worker service to handle background tasks:

- [ ] Finalize KeyGeneration handler
- [ ] Complete KeySign handlers
- [ ] Add proper error handling and retries for tasks
- [ ] Implement vault backup and recovery
- [ ] Add metrics and monitoring for task processing

Implementation notes:

- Worker tasks are defined in `internal/tasks/tasks.go`
- Handlers are in `internal/service/worker.go`
- Consider implementing task timeouts and circuit breakers

## Implementation Timeline

### Week 1: Core Cryptography and API Foundations

1. **Days 1-2: TSS Integration**

   - Set up basic TSS integration
   - Implement key generation flow
2. **Days 3-4: Basic API Completion**

   - Complete vault creation endpoint
   - Implement signing endpoint
   - Add signature verification
3. **Days 5: Testing and Documentation**

   - Test basic cryptographic operations
   - Document TSS integration

### Week 2: API Completion and Security

1. **Days 1-2: Policy Management**

   - Complete policy CRUD operations
   - Add policy validation
2. **Days 3-4: Authentication**

   - Enhance authentication service
   - Implement proper JWT handling
3. **Day 5: Transaction Management**

   - Complete transaction handling
   - Add transaction history

### Week 3: Worker and Finalization

1. **Days 1-2: Worker Tasks**

   - Complete worker task handlers
   - Add error handling and retries
2. **Days 3-4: Testing**

   - Add comprehensive tests
   - Test integration with plugins
3. **Day 5: Documentation and Cleanup**

   - Complete API documentation
   - Clean up code and dependencies

## Technical Details

### TSS Implementation Notes

The TSS integration should follow these patterns:

```go
// Example key generation flow
func HandleKeyGeneration(task *asynq.Task) error {
    var req types.VaultCreateRequest
    if err := json.Unmarshal(task.Payload(), &req); err != nil {
        return err
    }
  
    // Create TSS parameters
    params := tss.NewDKLSKeygenParams(
        req.Parties,
        req.LocalPartyId,
        req.HexChainCode,
    )
  
    // Execute key generation
    result, err := tss.ExecuteKeygeneration(params)
    if err != nil {
        return fmt.Errorf("key generation failed: %w", err)
    }
  
    // Store the vault
    vault := types.Vault{
        PublicKey: result.PublicKey,
        // other fields...
    }
  
    // Encrypt and save vault
    // ...
}
```

### Database Schema Enhancements

Consider adding these database enhancements:

1. Indexes for performance

```sql
-- Add to init schema
CREATE INDEX IF NOT EXISTS idx_policies_public_key ON policies (public_key);
CREATE INDEX IF NOT EXISTS idx_transactions_policy_id ON transactions (policy_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions (status);
```

2. Add transaction status types

```go
// Add to types/transaction.go
const (
    TxStatusPending   = "pending"
    TxStatusCompleted = "completed"
    TxStatusFailed    = "failed"
    TxStatusRejected  = "rejected"
)
```

### API Response Standardization

Standardize API responses with this structure:

```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// Usage
func (s *Server) GetVault(c echo.Context) error {
    // ... implementation
    if err != nil {
        return c.JSON(http.StatusInternalServerError, APIResponse{
            Success: false,
            Error:   err.Error(),
        })
    }
  
    return c.JSON(http.StatusOK, APIResponse{
        Success: true,
        Data:    vault,
    })
}
```

## Deployment Considerations

1. **Database Migrations**

   - Create a simple migration framework
   - Add version tracking for schema changes
2. **Configuration Management**

   - Add environment variable overrides for all config values
   - Create separate configs for dev, test, and prod
3. **Monitoring**

   - Add Prometheus metrics for key operations
   - Set up health check endpoints
   - Add structured logging
4. **Security**

   - Regular dependency updates
   - Security scanning in CI/CD pipeline
   - Proper secrets management
