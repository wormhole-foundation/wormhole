type ISOString = string;
const formatQuorumDate = (date: string): ISOString | string => {
  if (date.includes(" +0000 UTC")) {
    const formatted = date.replace(" +0000 UTC", "Z").replace(" ", "T");
    return formatted;
  }
  return date;
};

export { formatQuorumDate };
