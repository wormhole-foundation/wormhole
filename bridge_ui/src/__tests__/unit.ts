import { describe, expect, it } from "@jest/globals";
import { balancePretty } from "../utils/balancePretty";

describe("Unit Tests", () => {
  describe("balancePretty() tests", () => {
    it("9.99 => 9.99", () => {
      expect(balancePretty("9.99")).toBe("9.99");
    });
    it("123456.789 => 123456.78", () => {
      expect(balancePretty("123456.789")).toBe("123456.7");
    });
    it("1234567.8912 => 1.23 M", () => {
      expect(balancePretty("1234567.891")).toBe("1.23 M");
    });
    it("123999.8912 => 1.23 M", () => {
      expect(balancePretty("1239999.8912")).toBe("1.23 M");
    });
    it("981234567.8912 => 981.23 M", () => {
      expect(balancePretty("981234567.891")).toBe("981.23 M");
    });
    it("9876543210.8912 => 9.87 B", () => {
      expect(balancePretty("9876543210.8912")).toBe("9.87 B");
    });
    it("219876543210.8912 => 219.87 B", () => {
      expect(balancePretty("219876543210.8912")).toBe("219.87 B");
    });
    it("219876543210 => 219.87 B", () => {
      expect(balancePretty("219876543210")).toBe("219.87 B");
    });
  });
});
