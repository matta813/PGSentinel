export const operatorControlsNow = new Date();

export const localDateTime = (date: Date) =>
  new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
