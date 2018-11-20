# 6.0.1 / 2018-10-17

- Better error handling for errors thrown by plugins ([#242](https://github.com/postcss/postcss-cli/issues/242), [#243](https://github.com/postcss/postcss-cli/pull/243))
- Update dependencies
- Clarify docs ([#233](https://github.com/postcss/postcss-cli/issues/233))

# 6.0.0 / 2018-07-18

- Drop support for Node 4
- Upgrade to postcss v7 ([release notes](https://github.com/postcss/postcss/blob/master/CHANGELOG.md#70-president-amy))
- Upgrade to postcss-load-config v2 ([release notes](https://github.com/michael-ciniawsky/postcss-load-config/blob/master/CHANGELOG.md#200-2018-07-10))

# 5.0.1 / 2018-06-18

- Shallow copy options object; fixes a few edge cases
- Adjust options for file watching to play better with some editors

# 5.0.0 / 2018-02-06

- Now allows passing a directory as the input (all files in the directory will be processed)
- The CLI is now silent by default; added `--verbose` flag for if you want noisy logs
- Doesn't exit watch mode when there's an error in the plugin chain
- Removed non-obvious shorthand arguments (`-x`, `-p`, `-s`, `-t`, `-e`, `-b`, & `-c`). Also removed `-v` as an alias for `--version`.
- Prevent stupid option combinations like `--dir` & `-o`, and `--watch` & `--replace`
- Doesn't allow `--watch` when writing to STDOUT

# 4.1.1 / 2017-08-17

- Fixed bug with `--config`
- Upgraded dependencies

# 4.1.0 / 2017-06-10

- Can now pass a number to `--poll` to set poll interval
- Updated `postcss-reporter` dependency to v4.0.0

# 4.0.0 / 2017-05-09

- **BREAKING:** Upgrade postcss to v6.x

# 3.2.0 / 2017-04-21

- Added `--base` CLI option for keeping directory structure

# 3.1.1 / 2017-04-04

- Fixed `files` property in `package.json`; `lib/` folder wasn't included in v3.1.0

# 3.1.0 / 2017-04-04

- Improved incremental rebuilds for better performance in watch mode.
- Switched to `read-cache` for file reading for better performance.
- Set a dummy filename when reading from stdin to help plugins like autoprefixer find config files.
- Updated `fs-promise` dependency.

# 3.0.0 / 2017-03-15

## Changes since 3.0.0-beta

### Breaking Changes

- Don't exit on `CssSyntaxError` in watch mode. v2 behaved this way, but v3.0.0-beta didn't.
- Error out if `from` or `to` options are set in the config file. Use command line arguments instead.

### New Features

- Add `--poll` option. v2 had this, however, this new implementation removes the capability to set the interval, which was supported in v2.

### Bugfixes

- Set `from` option for correct sourcemaps
- Fix `--watch`'s glob handling
- Fix error handling

## Changes since v2.6.0

### Breaking Changes

- Uses https://github.com/michael-ciniawsky/postcss-load-config for config files. Dropped support for the v2 config file format.
- Can't set input files in config file; pass input files on the command line instead.
- `--use` accepts a list of plugins. This may cause issues if you have your list of css files at the end of your command.
- Can't pass options to plugins via `--plugin.key=value` anymore, use a config file.
- Changed usage of the `--map` option; use `--map` for external sourcemaps, `--no-map` to disable all maps. Inline sourcemaps are default.
- Removed `--log` flag; this behavior is now default.
- Removed the `--local-plugins` flag; same result can be achieved with `postcss.config.js`.
- Removed the global `watchCSS` handler, plugins that import/rely on other files should use a `dependency` message instead.
- Changed behavior of the `--poll` option; no longer accepts an integer `interval`.

### New Features

- `--ext` (`-x`) option allows you to set the file extensions for your output files when using `--dir`.
- `--env` allows you to set `NODE_ENV` in a cross-platform manner.

Migration guide for upgrading from v2: https://github.com/postcss/postcss-cli/wiki/Migrating-from-v2-to-v3

# 3.0.0-beta / 2017-03-17

## Breaking Changes

- Uses https://github.com/michael-ciniawsky/postcss-load-config for config files. Dropped support for the v2 config file format.
- Can't set input files in config file; pass input files on the command line instead.
- `--use` accepts a list of plugins. This may cause issues if you have your list of css files at the end of your command.
- Can't pass options to plugins via `--plugin.key=value` anymore, use a config file.
- Changed usage of the `--map` option; use `--map` for external sourcemaps, `--no-map` to disable all maps. Inline sourcemaps are default.
- Removed `--log` flag; this behavior is now default.
- Removed the `--local-plugins` flag; same result can be achieved with `postcss.config.js`.
- Removed the global `watchCSS` handler, plugins that import/rely on other files should use a `dependency` message instead.

## New Features

- `--ext` (`-x`) option allows you to set the file extensions for your output files when using `--dir`.
- `--env` allows you to set `NODE_ENV` in a cross-platform manner.

Migration guide: https://github.com/postcss/postcss-cli/wiki/Migrating-from-v2-to-v3

# 2.6.0 / 2016-08-30

- Add log option
- Update postcss-import to v8.1.2 from v7.1.0
- Update globby to v4.1.0 from v3.0.1
- Update postcss-url to v5.1.2 from v4.0.0
- Update jshint to v2.9.2 from v2.6.3
- Update chokidar to v1.5.1 from v1.0.3
- Update yargs to v4.7.1 from v3.32.0
- Support es6 export
- Allow running without plugins
- Add test for --poll
- Add --poll flag

# 2.5.2 / 2016-04-18

- Fix typo in help message: -use => [--use|-u]
- npm install --save mkdirp
- Support mkdirp to create dest path if it doesn't exists
- Fix booleans in config file

# 2.5.1 / 2016-02-11

- fix `input` argument

# 2.5.0 / 2016-01-30

- move to postcss/postcss-cli repository
- Update Readme.md

# 2.4.1 / 2016-01-27

- improve warning disply format

# 2.4.0 / 2016-01-15

- add support for source maps

# 2.3.3 / 2015-12-28

- add usage example for `local-plugins` option in config file

# 2.3.2 / 2015-10-27

- auto-configure postcss-import support
- add support for watching multiple entry points

# 2.3.1 / 2015-10-25

- update Travis config
- upgrade postcss-import dependency - fix deprecation warnings during make test-watch

# 2.3.0 / 2015-10-24

- add --local-plugins option that lets postcss-cli to look for plugins in current directory

# 2.2.0 / 2015-10-09

- add support for --replace|-r - if used input files are replaced with generated output
- refactor support for custom syntax options

# 2.1.1 / 2015-10-08

- add globby to support wildcards in Windows
- remove obsolete note on postcss-import compatibility

# 2.1.0 / 2015-09-01

- add support for PostCSS 5.0 custom syntax options

# 2.0.0 / 2015-08-24

- remove support for --safe option
- switch to using postcss 5.x

# 1.5.0 / 2015-07-20

- add watch mode (-w|--watch) in which postcss-cli observes and recompiles inputs whenever they change
- update neo-async dependency to released version
- update postcss-url dependency (used in tests only)

# 1.4.0 / 2015-07-12

- allow specifying input file via config file
- allow specifying -u|--use via config file

# 1.3.1 / 2015-05-03

- update npm keyword: postcssrunner -> postcss-runner

# 1.3.0 / 2015-04-28

- add support for stdin/stdout if no input/output file specified

# 1.2.1 / 2015-04-20

- fix typo in readme

# 1.2.0 / 2015-04-02

- display warnings and errors
- stop testing on node 0.10

# 1.1.0 / 2015-03-28

- prefer postcss async API if available

# 1.0.0 / 2015-03-22

- use official yargs version
- add support for multiple input files

# 0.3.0 / 2015-03-19

- support JS format as plugins config

# 0.2.0 / 2015-03-13

- use autoprefixer instead of autoprefixer-core
- change short options for --use from `p` to `u`
- add -v|--version support
- add --safe option to enable postcss safe mode

# 0.1.0 / 2015-03-11

- initial implementaion
