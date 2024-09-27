/* Use this file to export your Markdoc functions */

export const includes = {
  transform(parameters: Record<string, unknown>) {
    const [array, value] = Object.values(parameters);

    return Array.isArray(array) ? array.includes(value) : false;
  },
};

export const upper = {
  transform(parameters: unknown[]) {
    const string = parameters[0];
    return typeof string === "string" ? string.toUpperCase() : string;
  },
};
