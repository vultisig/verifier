export const fieldIsEmpty = (val?: string) => !val || !val.trim();

export const validateField = (
  paramName: string,
  value: string,
  required: boolean
): string => {
  if (required && fieldIsEmpty(value)) return `${paramName} is required`;
  return "";
};