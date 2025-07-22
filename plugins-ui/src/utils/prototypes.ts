declare global {
  interface String {
    capitalizeFirst(): string;
  }
}

String.prototype.capitalizeFirst = function () {
  return this.length ? this.charAt(0).toUpperCase() + this.slice(1) : `${this}`;
};
