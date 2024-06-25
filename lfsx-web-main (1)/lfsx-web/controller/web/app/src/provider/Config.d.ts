// This file ads TypeScript support for constants exported by the go program.
declare const Config: IConfig

interface IConfig {
    prod: boolean
}