import yargs, { CommandModule } from "yargs";

export class Yargs {
  yargs: typeof yargs;

  constructor(y: typeof yargs) {
    this.yargs = y;
  }

  addCommands = (addCommandsFn: YargsAddCommandsFn) => {
    this.yargs = addCommandsFn(this.yargs);
    return this;
  };

  y = () => this.yargs;
}

export type YargsAddCommandsFn = (y: typeof yargs) => typeof yargs;

export type YargsCommandModule = CommandModule<any, any>;
