import path from "path";
import { fileURLToPath } from "url";
import HtmlWebpackPlugin from "html-webpack-plugin";
import MiniCssExtractPlugin from "mini-css-extract-plugin";
import Dotenv from "dotenv-webpack";
import { BundleAnalyzerPlugin } from "webpack-bundle-analyzer";
import CssMinimizerPlugin from "css-minimizer-webpack-plugin";
import TerserPlugin from "terser-webpack-plugin";
import type { Configuration } from "webpack";
import "webpack-dev-server";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isProduction = process.env.NODE_ENV === "production";
const analyze = process.env.ANALYZE === "true";

const config: Configuration = {
  mode: isProduction ? "production" : "development",
  entry: "./src/index.tsx",
  output: {
    path: path.resolve(__dirname, "dist"),
    filename: isProduction ? "js/[name].[contenthash:8].js" : "js/[name].js",
    chunkFilename: isProduction
      ? "js/[name].[contenthash:8].chunk.js"
      : "js/[name].chunk.js",
    publicPath: "/",
    clean: true,
  },
  resolve: {
    extensions: [".tsx", ".ts", ".js", ".jsx"],
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  devtool: isProduction ? "source-map" : "eval-cheap-module-source-map",
  optimization: {
    minimize: isProduction,
    minimizer: [
      new TerserPlugin({
        parallel: true,
        terserOptions: {
          compress: {
            drop_console: true,
          },
        },
      }),
      new CssMinimizerPlugin(),
    ],
    splitChunks: {
      chunks: "all",
      name: false,
      cacheGroups: {
        vendor: {
          test: /[\\/]node_modules[\\/]/,
          name: "vendors",
          chunks: "all",
        },
      },
    },
  },
  module: {
    rules: [
      {
        test: /\.(ts|tsx|js|jsx)$/,
        exclude: /(node_modules)/,
        use: {
          loader: "swc-loader",
          options: {
            jsc: {
              parser: {
                syntax: "typescript",
                tsx: true,
                dynamicImport: true,
              },
              transform: {
                react: {
                  runtime: "automatic",
                },
              },
            },
          },
        },
      },
      {
        test: /\.css$/i,
        use: [
          isProduction ? MiniCssExtractPlugin.loader : "style-loader",
          "css-loader",
          "postcss-loader",
        ],
      },
      {
        test: /\.(png|jpe?g|gif|svg|woff2?|eot|ttf|otf)$/i,
        type: "asset",
        parser: {
          dataUrlCondition: {
            maxSize: 8 * 1024,
          },
        },
        generator: {
          filename: "assets/[hash][ext][query]",
        },
      },
    ],
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: "./public/index.html",
      minify: isProduction
        ? {
            removeComments: true,
            collapseWhitespace: true,
            removeRedundantAttributes: true,
            useShortDoctype: true,
            removeEmptyAttributes: true,
            removeStyleLinkTypeAttributes: true,
            keepClosingSlash: true,
            minifyJS: true,
            minifyCSS: true,
            minifyURLs: true,
          }
        : false,
    }),
    new Dotenv({
      systemvars: true,
    }),
    ...(isProduction
      ? [
          new MiniCssExtractPlugin({
            filename: "css/[name].[contenthash:8].css",
            chunkFilename: "css/[id].[contenthash:8].css",
          }),
        ]
      : []),
    ...(analyze ? [new BundleAnalyzerPlugin()] : []),
  ],
  devServer: {
    historyApiFallback: true,
    port: 3000,
    hot: true,
    compress: false,
    proxy: [
      {
        context: ["/api"],
        target: process.env.API_URL || "http://localhost:9090",
        changeOrigin: true,
        logLevel: "debug",
        onProxyReq: (proxyReq, req) => {
          console.log(
            `[Proxy] ${req.method} ${req.url} -> ${proxyReq.protocol}//${proxyReq.getHeader("host")}${proxyReq.path}`,
          );
        },
        onProxyRes: (proxyRes) => {
          // Disable buffering for SSE
          proxyRes.headers["x-accel-buffering"] = "no";
        },
      },
    ],
  },
};

export default config;
