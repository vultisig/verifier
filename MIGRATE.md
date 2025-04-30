# Vultisig Verifier Migration Guide

This document outlines the migration of the Vultisig Verifier component from the original monolithic repository to a standalone service.

## Overview

The Vultisig Verifier is responsible for:

- Processing vault creation and key generation
- Validating and signing transactions
- Managing plugin policies
- Providing cryptographic services for the Vultisig ecosystem

## Components Migrated

### Core Structure

- [X] Basic project structure following Go best practices
- [X] Configuration management with Viper
- [X] Command-line entrypoints (server and worker)
- [X] Docker configuration
- [X] Makefile and build scripts

### Data Models

- [X] Vault types (DKLS support)
- [X] Policy types
- [X] Transaction history
- [X] User authentication models

### Storage Layer

- [X] PostgreSQL implementation for database operations
- [X] Redis implementation for caching and task queue
- [X] S3-compatible storage for vault backups

### Service Layer

- [X] Authentication service (JWT-based)
- [X] Basic policy management service
- [X] Skeleton worker service

### API Layer

- [X] Web server setup with Echo framework
- [X] API route definitions
- [X] Basic middleware configuration
- [X] Endpoint stubs for all required functionality

### Cryptographic Utilities

- [X] Basic cryptographic helper functions
- [X] Placeholder signature verification
- [X] Vault encryption/decryption utilities

## Components Needing Implementation

### Cryptographic Integration

- [ ] Complete TSS library integration for DKLS
- [ ] Implement actual signing logic
- [ ] Wire up signature verification

### API Endpoints

- [ ] Complete implementation of `SignMessages` endpoint
- [ ] Complete implementation of `ReshareVault` endpoint
- [ ] Complete implementation of policy management endpoints
- [ ] Complete implementation of transaction endpoints

### Worker Tasks

- [ ] Finalize key generation task processing
- [ ] Implement signing task processing
- [ ] Add proper error handling and retry logic

### Database Operations

- [ ] Add indexes for better performance
- [ ] Implement database migrations
- [ ] Add comprehensive database testing

### Security Enhancements

- [ ] Implement comprehensive authentication
- [ ] Add authorization rules
- [ ] Enhance JWT implementation with proper token management

### Testing

- [ ] Add unit tests for all components
- [ ] Add integration tests
- [ ] Create CI/CD pipeline for automated testing

## Migration Steps

1. **Initial Setup**

   - [X] Copy core structure to new repository
   - [X] Update module paths and imports
   - [X] Verify build and basic functionality
2. **Database Migration**

   - [ ] Create schema and migration scripts
   - [ ] Test data migration process
   - [ ] Create backup strategy
3. **Service Implementation**

   - [ ] Complete each endpoint implementation
   - [ ] Implement proper error handling
   - [ ] Add metrics and monitoring
4. **Testing & Validation**

   - [ ] Test each endpoint
   - [ ] Verify compatibility with existing plugins
   - [ ] Performance testing
5. **Deployment**

   - [ ] Create deployment configuration
   - [ ] Set up monitoring
   - [ ] Plan cutover strategy

## Next Steps

1. **Immediate Actions**

   - Complete implementation of the cryptographic operations with TSS library
   - Finish implementation of all API endpoints
   - Add comprehensive error handling
2. **Follow-up Tasks**

   - Add automated tests
   - Implement metrics and monitoring
   - Create deployment documentation
3. **Final Migration**

   - Test with production-like data
   - Plan cutover with minimal downtime
   - Update client applications to use new service

## Technical Considerations

### Module Dependencies

The verifier relies on several key dependencies:

- Echo framework for web service
- Asynq for task queue
- PostgreSQL for database storage
- Redis for caching and queuing
- AWS S3 (or compatible) for block storage

### Configuration Changes

The configuration structure has been simplified and focused solely on verifier requirements:

- Server configuration
- Redis configuration
- Block storage configuration
- JWT authentication

### Integration Points

The verifier interacts with the plugin system through:

- Policy synchronization
- Transaction signing requests
- Vault creation and management

When completing the migration, care must be taken to maintain compatibility with these integration points.
