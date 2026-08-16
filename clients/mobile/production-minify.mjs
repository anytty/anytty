export const productionMinify = Object.freeze({
  compress: Object.freeze({
    dropConsole: true,
  }),
  mangle: true,
  codegen: true,
})

const releaseConsoleSink = 'const console=/*@__PURE__*/Object.freeze({debug(){},error(){},info(){},log(){},trace(){},warn(){}});'

export function productionRolldownOutput() {
  return {
    intro: releaseConsoleSink,
    minify: productionMinify,
  }
}
