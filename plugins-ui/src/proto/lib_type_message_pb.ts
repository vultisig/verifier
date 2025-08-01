// @generated by protoc-gen-es v2.4.0 with parameter "target=ts"
// @generated from file vultisig/keygen/v1/lib_type_message.proto (package vultisig.keygen.v1, syntax proto3)
 

import type { GenEnum, GenFile } from "@bufbuild/protobuf/codegenv1";
import { enumDesc, fileDesc } from "@bufbuild/protobuf/codegenv1";

/**
 * Describes the file vultisig/keygen/v1/lib_type_message.proto.
 */
export const file_vultisig_keygen_v1_lib_type_message: GenFile = /*@__PURE__*/
  fileDesc("Cil2dWx0aXNpZy9rZXlnZW4vdjEvbGliX3R5cGVfbWVzc2FnZS5wcm90bxISdnVsdGlzaWcua2V5Z2VuLnYxKi8KB0xpYlR5cGUSEQoNTElCX1RZUEVfR0cyMBAAEhEKDUxJQl9UWVBFX0RLTFMQAUJSChJ2dWx0aXNpZy5rZXlnZW4udjFaN2dpdGh1Yi5jb20vdnVsdGlzaWcvY29tbW9uZGF0YS9nby92dWx0aXNpZy9rZXlnZW4vdjE7djG6AgJWU2IGcHJvdG8z");

/**
 * buf:lint:ignore ENUM_ZERO_VALUE_SUFFIX
 *
 * @generated from enum vultisig.keygen.v1.LibType
 */
export enum LibType {
  /**
   * Default to GG20
   *
   * @generated from enum value: LIB_TYPE_GG20 = 0;
   */
  GG20 = 0,

  /**
   * @generated from enum value: LIB_TYPE_DKLS = 1;
   */
  DKLS = 1,
}

/**
 * Describes the enum vultisig.keygen.v1.LibType.
 */
export const LibTypeSchema: GenEnum<LibType> = /*@__PURE__*/
  enumDesc(file_vultisig_keygen_v1_lib_type_message, 0);

