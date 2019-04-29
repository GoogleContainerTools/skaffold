const path = require( 'path' );
const HtmlWebpackPlugin = require( 'html-webpack-plugin' );

module.exports = {
	entry: './src/main.js',
	output: {
		path: path.join( __dirname, '/dist' ),
		filename: 'main.js'
	},
	devServer:{
		host: '0.0.0.0'
	},
	module: {
		rules: [
			{
				test: /\.js$/,
				exclude: /node_modules/,
				use: [ 'babel-loader' ]
			},
			{
				test: /\.css$/,
				use: [ 'style-loader', 'css-loader' ]
			}
		]
	},
	plugins: [ new HtmlWebpackPlugin( { template: './src/index.html' } ) ]
};
