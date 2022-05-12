export const balancePretty = (uiString: string) => {
  const numberString = uiString.split(".")[0];
  const nsLen = numberString.length;
  if (nsLen > 9) {
    // Billion case
    const num = numberString.substring(0, nsLen - 9);
    const fract = numberString.substring(nsLen - 9, nsLen - 9 + 2);
    return num + "." + fract + " B";
  } else if (nsLen > 6) {
    // Million case
    const num = numberString.substring(0, nsLen - 6);
    const fract = numberString.substring(nsLen - 6, nsLen - 6 + 2);
    return num + "." + fract + " M";
  } else if (uiString.length > 8) {
    return uiString.substring(0, 8);
  } else {
    return uiString;
  }
};
