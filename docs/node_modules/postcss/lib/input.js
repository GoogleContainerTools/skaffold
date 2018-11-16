"use strict";

exports.__esModule = true;
exports.default = void 0;

var _cssSyntaxError = _interopRequireDefault(require("./css-syntax-error"));

var _previousMap = _interopRequireDefault(require("./previous-map"));

var _path = _interopRequireDefault(require("path"));

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function _defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ("value" in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } }

function _createClass(Constructor, protoProps, staticProps) { if (protoProps) _defineProperties(Constructor.prototype, protoProps); if (staticProps) _defineProperties(Constructor, staticProps); return Constructor; }

var sequence = 0;
/**
 * Represents the source CSS.
 *
 * @example
 * const root  = postcss.parse(css, { from: file })
 * const input = root.source.input
 */

var Input =
/*#__PURE__*/
function () {
  /**
   * @param {string} css    Input CSS source.
   * @param {object} [opts] {@link Processor#process} options.
   */
  function Input(css, opts) {
    if (opts === void 0) {
      opts = {};
    }

    if (css === null || typeof css === 'object' && !css.toString) {
      throw new Error("PostCSS received " + css + " instead of CSS string");
    }
    /**
     * Input CSS source
     *
     * @type {string}
     *
     * @example
     * const input = postcss.parse('a{}', { from: file }).input
     * input.css //=> "a{}"
     */


    this.css = css.toString();

    if (this.css[0] === "\uFEFF" || this.css[0] === "\uFFFE") {
      this.css = this.css.slice(1);
    }

    if (opts.from) {
      if (/^\w+:\/\//.test(opts.from)) {
        /**
         * The absolute path to the CSS source file defined
         * with the `from` option.
         *
         * @type {string}
         *
         * @example
         * const root = postcss.parse(css, { from: 'a.css' })
         * root.source.input.file //=> '/home/ai/a.css'
         */
        this.file = opts.from;
      } else {
        this.file = _path.default.resolve(opts.from);
      }
    }

    var map = new _previousMap.default(this.css, opts);

    if (map.text) {
      /**
       * The input source map passed from a compilation step before PostCSS
       * (for example, from Sass compiler).
       *
       * @type {PreviousMap}
       *
       * @example
       * root.source.input.map.consumer().sources //=> ['a.sass']
       */
      this.map = map;
      var file = map.consumer().file;
      if (!this.file && file) this.file = this.mapResolve(file);
    }

    if (!this.file) {
      sequence += 1;
      /**
       * The unique ID of the CSS source. It will be created if `from` option
       * is not provided (because PostCSS does not know the file path).
       *
       * @type {string}
       *
       * @example
       * const root = postcss.parse(css)
       * root.source.input.file //=> undefined
       * root.source.input.id   //=> "<input css 1>"
       */

      this.id = '<input css ' + sequence + '>';
    }

    if (this.map) this.map.file = this.from;
  }

  var _proto = Input.prototype;

  _proto.error = function error(message, line, column, opts) {
    if (opts === void 0) {
      opts = {};
    }

    var result;
    var origin = this.origin(line, column);

    if (origin) {
      result = new _cssSyntaxError.default(message, origin.line, origin.column, origin.source, origin.file, opts.plugin);
    } else {
      result = new _cssSyntaxError.default(message, line, column, this.css, this.file, opts.plugin);
    }

    result.input = {
      line: line,
      column: column,
      source: this.css
    };
    if (this.file) result.input.file = this.file;
    return result;
  };
  /**
   * Reads the input source map and returns a symbol position
   * in the input source (e.g., in a Sass file that was compiled
   * to CSS before being passed to PostCSS).
   *
   * @param {number} line   Line in input CSS.
   * @param {number} column Column in input CSS.
   *
   * @return {filePosition} Position in input source.
   *
   * @example
   * root.source.input.origin(1, 1) //=> { file: 'a.css', line: 3, column: 1 }
   */


  _proto.origin = function origin(line, column) {
    if (!this.map) return false;
    var consumer = this.map.consumer();
    var from = consumer.originalPositionFor({
      line: line,
      column: column
    });
    if (!from.source) return false;
    var result = {
      file: this.mapResolve(from.source),
      line: from.line,
      column: from.column
    };
    var source = consumer.sourceContentFor(from.source);
    if (source) result.source = source;
    return result;
  };

  _proto.mapResolve = function mapResolve(file) {
    if (/^\w+:\/\//.test(file)) {
      return file;
    }

    return _path.default.resolve(this.map.consumer().sourceRoot || '.', file);
  };
  /**
   * The CSS source identifier. Contains {@link Input#file} if the user
   * set the `from` option, or {@link Input#id} if they did not.
   *
   * @type {string}
   *
   * @example
   * const root = postcss.parse(css, { from: 'a.css' })
   * root.source.input.from //=> "/home/ai/a.css"
   *
   * const root = postcss.parse(css)
   * root.source.input.from //=> "<input css 1>"
   */


  _createClass(Input, [{
    key: "from",
    get: function get() {
      return this.file || this.id;
    }
  }]);

  return Input;
}();

var _default = Input;
/**
 * @typedef  {object} filePosition
 * @property {string} file   Path to file.
 * @property {number} line   Source line in file.
 * @property {number} column Source column in file.
 */

exports.default = _default;
module.exports = exports.default;
//# sourceMappingURL=data:application/json;charset=utf8;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImlucHV0LmVzNiJdLCJuYW1lcyI6WyJzZXF1ZW5jZSIsIklucHV0IiwiY3NzIiwib3B0cyIsInRvU3RyaW5nIiwiRXJyb3IiLCJzbGljZSIsImZyb20iLCJ0ZXN0IiwiZmlsZSIsInBhdGgiLCJyZXNvbHZlIiwibWFwIiwiUHJldmlvdXNNYXAiLCJ0ZXh0IiwiY29uc3VtZXIiLCJtYXBSZXNvbHZlIiwiaWQiLCJlcnJvciIsIm1lc3NhZ2UiLCJsaW5lIiwiY29sdW1uIiwicmVzdWx0Iiwib3JpZ2luIiwiQ3NzU3ludGF4RXJyb3IiLCJzb3VyY2UiLCJwbHVnaW4iLCJpbnB1dCIsIm9yaWdpbmFsUG9zaXRpb25Gb3IiLCJzb3VyY2VDb250ZW50Rm9yIiwic291cmNlUm9vdCJdLCJtYXBwaW5ncyI6Ijs7Ozs7QUFBQTs7QUFDQTs7QUFFQTs7Ozs7Ozs7QUFFQSxJQUFJQSxRQUFRLEdBQUcsQ0FBZjtBQUVBOzs7Ozs7OztJQU9NQyxLOzs7QUFDSjs7OztBQUlBLGlCQUFhQyxHQUFiLEVBQWtCQyxJQUFsQixFQUE4QjtBQUFBLFFBQVpBLElBQVk7QUFBWkEsTUFBQUEsSUFBWSxHQUFMLEVBQUs7QUFBQTs7QUFDNUIsUUFBSUQsR0FBRyxLQUFLLElBQVIsSUFBaUIsT0FBT0EsR0FBUCxLQUFlLFFBQWYsSUFBMkIsQ0FBQ0EsR0FBRyxDQUFDRSxRQUFyRCxFQUFnRTtBQUM5RCxZQUFNLElBQUlDLEtBQUosdUJBQStCSCxHQUEvQiw0QkFBTjtBQUNEO0FBRUQ7Ozs7Ozs7Ozs7O0FBU0EsU0FBS0EsR0FBTCxHQUFXQSxHQUFHLENBQUNFLFFBQUosRUFBWDs7QUFFQSxRQUFJLEtBQUtGLEdBQUwsQ0FBUyxDQUFULE1BQWdCLFFBQWhCLElBQTRCLEtBQUtBLEdBQUwsQ0FBUyxDQUFULE1BQWdCLFFBQWhELEVBQTBEO0FBQ3hELFdBQUtBLEdBQUwsR0FBVyxLQUFLQSxHQUFMLENBQVNJLEtBQVQsQ0FBZSxDQUFmLENBQVg7QUFDRDs7QUFFRCxRQUFJSCxJQUFJLENBQUNJ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Z0JULElBQUksQ0FBQ2tCLE1BQXJCLENBREs7QUFFWEwsTUFBQUEsSUFBSSxFQUFFYixJQUFJLENBQUNhLElBRkE7QUFHWEMsTUFBQUEsTUFBTSxFQUFFZCxJQUFJLENBQUNjO0FBSEYsS0FBYjtBQU1BLFFBQUlJLE1BQU0sR0FBR1YsUUFBUSxDQUFDYyxnQkFBVCxDQUEwQnRCLElBQUksQ0FBQ2tCLE1BQS9CLENBQWI7QUFDQSxRQUFJQSxNQUFKLEVBQVlILE1BQU0sQ0FBQ0csTUFBUCxHQUFnQkEsTUFBaEI7QUFFWixXQUFPSCxNQUFQO0FBQ0QsRzs7U0FFRE4sVSx1QkFBWVAsSSxFQUFNO0FBQ2hCLFFBQUksWUFBWUQsSUFBWixDQUFpQkMsSUFBakIsQ0FBSixFQUE0QjtBQUMxQixhQUFPQSxJQUFQO0FBQ0Q7O0FBQ0QsV0FBT0MsY0FBS0MsT0FBTCxDQUFhLEtBQUtDLEdBQUwsQ0FBU0csUUFBVCxHQUFvQmUsVUFBcEIsSUFBa0MsR0FBL0MsRUFBb0RyQixJQUFwRCxDQUFQO0FBQ0QsRztBQUVEOzs7Ozs7Ozs7Ozs7Ozs7Ozt3QkFhWTtBQUNWLGFBQU8sS0FBS0EsSUFBTCxJQUFhLEtBQUtRLEVBQXpCO0FBQ0Q7Ozs7OztlQUdZaEIsSztBQUVmIiwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IENzc1N5bnRheEVycm9yIGZyb20gJy4vY3NzLXN5bnRheC1lcnJvcidcbmltcG9ydCBQcmV2aW91c01hcCBmcm9tICcuL3ByZXZpb3VzLW1hcCdcblxuaW1wb3J0IHBhdGggZnJvbSAncGF0aCdcblxubGV0IHNlcXVlbmNlID0gMFxuXG4vKipcbiAqIFJlcHJlc2VudHMgdGhlIHNvdXJjZSBDU1MuXG4gKlxuICogQGV4YW1wbGVcbiAqIGNvbnN0IHJvb3QgID0gcG9zdGNzcy5wYXJzZShjc3MsIHsgZnJvbTogZmlsZSB9KVxuICogY29uc3QgaW5wdXQgPSByb290LnNvdXJjZS5pbnB1dFxuICovXG5jbGFzcyBJbnB1dCB7XG4gIC8qKlxuICAgKiBAcGFyYW0ge3N0cmluZ30gY3NzICAgIElucHV0IENTUyBzb3VyY2UuXG4gICAqIEBwYXJhbSB7b2JqZWN0fSBbb3B0c10ge0BsaW5rIFByb2Nlc3NvciNwcm9jZXNzfSBvcHRpb25zLlxuICAgKi9cbiAgY29uc3RydWN0b3IgKGNzcywgb3B0cyA9IHsgfSkge1xuICAgIGlmIChjc3MgPT09IG51bGwgfHwgKHR5cGVvZiBjc3MgPT09ICdvYmplY3QnICYmICFjc3MudG9TdHJpbmcpKSB7XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoYFBvc3RDU1MgcmVjZWl2ZWQgJHsgY3NzIH0gaW5zdGVhZCBvZiBDU1Mgc3RyaW5nYClcbiAgICB9XG5cbiAgICAvKipcbiAgICAgKiBJbnB1dCBDU1Mgc291cmNlXG4gICAgICpcbiAgICAgKiBAdHlwZSB7c3RyaW5nfVxuICAgICAqXG4gICAgICogQGV4YW1wbGVcbiAgICAgKiBjb25zdCBpbnB1dCA9IHBvc3Rjc3MucGFyc2UoJ2F7fScsIHsgZnJvbTogZmlsZSB9KS5pbnB1dFxuICAgICAqIGlucHV0LmNzcyAvLz0+IFwiYXt9XCJcbiAgICAgKi9cbiAgICB0aGlzLmNzcyA9IGNzcy50b1N0cmluZygpXG5cbiAgICBpZiAodGhpcy5jc3NbMF0gPT09ICdcXHVGRUZGJyB8fCB0aGlzLmNzc1swXSA9PT0gJ1xcdUZGRkUnKSB7XG4gICAgICB0aGlzLmNzcyA9IHRoaXMuY3NzLnNsaWNlKDEpXG4gICAgfVxuXG4gICAgaWYgKG9wdHMuZnJvbSkge1xuICAgICAgaWYgKC9eXFx3KzpcXC9cXC8vLnRlc3Qob3B0cy5mcm9tKSkge1xuICAgICAgICAvKipcbiAgICAgICAgICogVGhlIGFic29sdXRlIHBhdGggdG8gdGhlIENTUyBzb3VyY2UgZmlsZSBkZWZpbmVkXG4gICAgICAgICAqIHdpdGggdGhlIGBmcm9tYCBvcHRpb24uXG4gICAgICAgICAqXG4gICAgICAgICAqIEB0eXBlIHtzdHJpbmd9XG4gICAgICAgICAqXG4gICAgICAgICAqIEBleGFtcGxlXG4gICAgICAgICAqIGNvbnN0IHJvb3QgPSBwb3N0Y3NzLnBhcnNlKGNzcywgeyBmcm9tOiAnYS5jc3MnIH0pXG4gICAgICAgICAqIHJvb3Quc291cmNlLmlucHV0LmZpbGUgLy89PiAnL2hvbWUvYWkvYS5jc3MnXG4gICAgICAgICAqL1xuICAgICAgICB0aGlzLmZpbGUgPSBvcHRzLmZyb21cbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHRoaXMuZmlsZSA9IHBhdGgucmVzb2x2ZShvcHRzLmZyb20pXG4gICAgICB9XG4gICAgfVxuXG4gICAgbGV0IG1hcCA9IG5ldyBQcmV2aW91c01hcCh0aGlzLmNzcywgb3B0cylcbiAgICBpZiAobWFwLnRleHQpIHtcbiAgICAgIC8qKlxuICAgICAgICogVGhlIGlucHV0IHNvdXJjZSBtYXAgcGFzc2VkIGZyb20gYSBjb21waWxhdGlvbiBzdGVwIGJlZm9yZSBQb3N0Q1NTXG4gICAgICAgKiAoZm9yIGV4YW1wbGUsIGZyb20gU2FzcyBjb21waWxlcikuXG4gICAgICAgKlxuICAgICAgICogQHR5cGUge1ByZXZpb3VzTWFwfVxuICAgICAgICpcbiAgICAgICAqIEBleGFtcGxlXG4gICAgICAgKiByb290LnNvdXJjZS5pbnB1dC5tYXAuY29uc3VtZXIoKS5zb3VyY2VzIC8vPT4gWydhLnNhc3MnXVxuICAgICAgICovXG4gICAgICB0aGlzLm1hcCA9IG1hcFxuICAgICAgbGV0IGZpbGUgPSBtYXAuY29uc3VtZXIoKS5maWxlXG4gICAgICBpZiAoIXRoaXMuZmlsZSAmJiBmaWxlKSB0aGlzLmZpbGUgPSB0aGlzLm1hcFJlc29sdmUoZmlsZSlcbiAgICB9XG5cbiAgICBpZiAoIXRoaXMuZmlsZSkge1xuICAgICAgc2VxdWVuY2UgKz0gMVxuICAgICAgLyoqXG4gICAgICAgKiBUaGUgdW5pcXVlIElEIG9mIHRoZSBDU1Mgc291cmNlLiBJdCB3aWxsIGJlIGNyZWF0ZWQgaWYgYGZyb21gIG9wdGlvblxuICAgICAgICogaXMgbm90IHByb3ZpZGVkIChiZWNhdXNlIFBvc3RDU1MgZG9lcyBub3Qga25vdyB0aGUgZmlsZSBwYXRoKS5cbiAgICAgICAqXG4gICAgICAgKiBAdHlwZSB7c3RyaW5nfVxuICAgICAgICpcbiAgICAgICAqIEBleGFtcGxlXG4gICAgICAgKiBjb25zdCByb290ID0gcG9zdGNzcy5wYXJzZShjc3MpXG4gICAgICAgKiByb290LnNvdXJjZS5pbnB1dC5maWxlIC8vPT4gdW5kZWZpbmVkXG4gICAgICAgKiByb290LnNvdXJjZS5pbnB1dC5pZCAgIC8vPT4gXCI8aW5wdXQgY3NzIDE+XCJcbiAgICAgICAqL1xuICAgICAgdGhpcy5pZCA9ICc8aW5wdXQgY3NzICcgKyBzZXF1ZW5jZSArICc+J1xuICAgIH1cbiAgICBpZiAodGhpcy5tYXApIHRoaXMubWFwLmZpbGUgPSB0aGlzLmZyb21cbiAgfVxuXG4gIGVycm9yIChtZXNzYWdlLCBsaW5lLCBjb2x1bW4sIG9wdHMgPSB7IH0pIHtcbiAgICBsZXQgcmVzdWx0XG4gICAgbGV0IG9yaWdpbiA9IHRoaXMub3JpZ2luKGxpbmUsIGNvbHVtbilcbiAgICBpZiAob3JpZ2luKSB7XG4gICAgICByZXN1bHQgPSBuZXcgQ3NzU3ludGF4RXJyb3IoXG4gICAgICAgIG1lc3NhZ2UsIG9yaWdpbi5saW5lLCBvcmlnaW4uY29sdW1uLFxuICAgICAgICBvcmlnaW4uc291cmNlLCBvcmlnaW4uZmlsZSwgb3B0cy5wbHVnaW5cbiAgICAgIClcbiAgICB9IGVsc2Uge1xuICAgICAgcmVzdWx0ID0gbmV3IENzc1N5bnRheEVycm9yKFxuICAgICAgICBtZXNzYWdlLCBsaW5lLCBjb2x1bW4sIHRoaXMuY3NzLCB0aGlzLmZpbGUsIG9wdHMucGx1Z2luKVxuICAgIH1cblxuICAgIHJlc3VsdC5pbnB1dCA9IHsgbGluZSwgY29sdW1uLCBzb3VyY2U6IHRoaXMuY3NzIH1cbiAgICBpZiAodGhpcy5maWxlKSByZXN1bHQuaW5wdXQuZmlsZSA9IHRoaXMuZmlsZVxuXG4gICAgcmV0dXJuIHJlc3VsdFxuICB9XG5cbiAgLyoqXG4gICAqIFJlYWRzIHRoZSBpbnB1dCBzb3VyY2UgbWFwIGFuZCByZXR1cm5zIGEgc3ltYm9sIHBvc2l0aW9uXG4gICAqIGluIHRoZSBpbnB1dCBzb3VyY2UgKGUuZy4sIGluIGEgU2FzcyBmaWxlIHRoYXQgd2FzIGNvbXBpbGVkXG4gICAqIHRvIENTUyBiZWZvcmUgYmVpbmcgcGFzc2VkIHRvIFBvc3RDU1MpLlxuICAgKlxuICAgKiBAcGFyYW0ge251bWJlcn0gbGluZSAgIExpbmUgaW4gaW5wdXQgQ1NTLlxuICAgKiBAcGFyYW0ge251bWJlcn0gY29sdW1uIENvbHVtbiBpbiBpbnB1dCBDU1MuXG4gICAqXG4gICAqIEByZXR1cm4ge2ZpbGVQb3NpdGlvbn0gUG9zaXRpb24gaW4gaW5wdXQgc291cmNlLlxuICAgKlxuICAgKiBAZXhhbXBsZVxuICAgKiByb290LnNvdXJjZS5pbnB1dC5vcmlnaW4oMSwgMSkgLy89PiB7IGZpbGU6ICdhLmNzcycsIGxpbmU6IDMsIGNvbHVtbjogMSB9XG4gICAqL1xuICBvcmlnaW4gKGxpbmUsIGNvbHVtbikge1xuICAgIGlmICghdGhpcy5tYXApIHJldHVybiBmYWxzZVxuICAgIGxldCBjb25zdW1lciA9IHRoaXMubWFwLmNvbnN1bWVyKClcblxuICAgIGxldCBmcm9tID0gY29uc3VtZXIub3JpZ2luYWxQb3NpdGlvbkZvcih7IGxpbmUsIGNvbHVtbiB9KVxuICAgIGlmICghZnJvbS5zb3VyY2UpIHJldHVybiBmYWxzZVxuXG4gICAgbGV0IHJlc3VsdCA9IHtcbiAgICAgIGZpbGU6IHRoaXMubWFwUmVzb2x2ZShmcm9tLnNvdXJjZSksXG4gICAgICBsaW5lOiBmcm9tLmxpbmUsXG4gICAgICBjb2x1bW46IGZyb20uY29sdW1uXG4gICAgfVxuXG4gICAgbGV0IHNvdXJjZSA9IGNvbnN1bWVyLnNvdXJjZUNvbnRlbnRGb3IoZnJvbS5zb3VyY2UpXG4gICAgaWYgKHNvdXJjZSkgcmVzdWx0LnNvdXJjZSA9IHNvdXJjZVxuXG4gICAgcmV0dXJuIHJlc3VsdFxuICB9XG5cbiAgbWFwUmVzb2x2ZSAoZmlsZSkge1xuICAgIGlmICgvXlxcdys6XFwvXFwvLy50ZXN0KGZpbGUpKSB7XG4gICAgICByZXR1cm4gZmlsZVxuICAgIH1cbiAgICByZXR1cm4gcGF0aC5yZXNvbHZlKHRoaXMubWFwLmNvbnN1bWVyKCkuc291cmNlUm9vdCB8fCAnLicsIGZpbGUpXG4gIH1cblxuICAvKipcbiAgICogVGhlIENTUyBzb3VyY2UgaWRlbnRpZmllci4gQ29udGFpbnMge0BsaW5rIElucHV0I2ZpbGV9IGlmIHRoZSB1c2VyXG4gICAqIHNldCB0aGUgYGZyb21gIG9wdGlvbiwgb3Ige0BsaW5rIElucHV0I2lkfSBpZiB0aGV5IGRpZCBub3QuXG4gICAqXG4gICAqIEB0eXBlIHtzdHJpbmd9XG4gICAqXG4gICAqIEBleGFtcGxlXG4gICAqIGNvbnN0IHJvb3QgPSBwb3N0Y3NzLnBhcnNlKGNzcywgeyBmcm9tOiAnYS5jc3MnIH0pXG4gICAqIHJvb3Quc291cmNlLmlucHV0LmZyb20gLy89PiBcIi9ob21lL2FpL2EuY3NzXCJcbiAgICpcbiAgICogY29uc3Qgcm9vdCA9IHBvc3Rjc3MucGFyc2UoY3NzKVxuICAgKiByb290LnNvdXJjZS5pbnB1dC5mcm9tIC8vPT4gXCI8aW5wdXQgY3NzIDE+XCJcbiAgICovXG4gIGdldCBmcm9tICgpIHtcbiAgICByZXR1cm4gdGhpcy5maWxlIHx8IHRoaXMuaWRcbiAgfVxufVxuXG5leHBvcnQgZGVmYXVsdCBJbnB1dFxuXG4vKipcbiAqIEB0eXBlZGVmICB7b2JqZWN0fSBmaWxlUG9zaXRpb25cbiAqIEBwcm9wZXJ0eSB7c3RyaW5nfSBmaWxlICAgUGF0aCB0byBmaWxlLlxuICogQHByb3BlcnR5IHtudW1iZXJ9IGxpbmUgICBTb3VyY2UgbGluZSBpbiBmaWxlLlxuICogQHByb3BlcnR5IHtudW1iZXJ9IGNvbHVtbiBTb3VyY2UgY29sdW1uIGluIGZpbGUuXG4gKi9cbiJdLCJmaWxlIjoiaW5wdXQuanMifQ==
