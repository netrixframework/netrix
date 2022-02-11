module.exports = {
    configureWebpack: config => {
        config.entry.app = ['./dashboard/main.js']
    },
    publicPath: "/dashboard/app",
    outputDir: "apiserver/dist"
}