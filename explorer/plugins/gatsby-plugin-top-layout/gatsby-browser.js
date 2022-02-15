import React from "react";
import TopLayout from "./TopLayout";

export const wrapRootElement = ({ element }) => (
  <TopLayout>{element}</TopLayout>
);
