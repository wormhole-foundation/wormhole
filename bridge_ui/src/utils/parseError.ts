const MM_ERR_WITH_INFO_START =
  "VM Exception while processing transaction: revert ";
const parseError = (e: any) =>
  e?.data?.message?.startsWith(MM_ERR_WITH_INFO_START)
    ? e.data.message.replace(MM_ERR_WITH_INFO_START, "")
    : e?.response?.data?.error // terra error
    ? e.response.data.error
    : e?.message
    ? e.message
    : "An unknown error occurred";
export default parseError;
