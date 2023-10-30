import type { JestConfigWithTsJest } from "ts-jest";

const jestConfig: JestConfigWithTsJest = {
    preset: "ts-jest",
    verbose: true,
    testPathIgnorePatterns: ["utils"],
    transform: {
        "^.+\\.tsx?$": [
            "ts-jest",
            {
                isolatedModules: true,
            },
        ],
    },
};

export default jestConfig;