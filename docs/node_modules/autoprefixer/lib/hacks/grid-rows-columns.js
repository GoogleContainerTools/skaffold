"use strict";

function _defaults(obj, defaults) { var keys = Object.getOwnPropertyNames(defaults); for (var i = 0; i < keys.length; i++) { var key = keys[i]; var value = Object.getOwnPropertyDescriptor(defaults, key); if (value && value.configurable && obj[key] === undefined) { Object.defineProperty(obj, key, value); } } return obj; }

function _inheritsLoose(subClass, superClass) { subClass.prototype = Object.create(superClass.prototype); subClass.prototype.constructor = subClass; _defaults(subClass, superClass); }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }

var Declaration = require('../declaration');

var _require = require('./grid-utils'),
    prefixTrackProp = _require.prefixTrackProp,
    prefixTrackValue = _require.prefixTrackValue;

var GridRowsColumns =
/*#__PURE__*/
function (_Declaration) {
  _inheritsLoose(GridRowsColumns, _Declaration);

  function GridRowsColumns() {
    return _Declaration.apply(this, arguments) || this;
  }

  var _proto = GridRowsColumns.prototype;

  /**
   * Change property name for IE
   */
  _proto.prefixed = function prefixed(prop, prefix) {
    if (prefix === '-ms-') {
      return prefixTrackProp({
        prop: prop,
        prefix: prefix
      });
    }

    return _Declaration.prototype.prefixed.call(this, prop, prefix);
  };
  /**
   * Change IE property back
   */


  _proto.normalize = function normalize(prop) {
    return prop.replace(/^grid-(rows|columns)/, 'grid-template-$1');
  };
  /**
   * Change repeating syntax for IE
   */


  _proto.set = function set(decl, prefix) {
    if (prefix === '-ms-' && decl.value.indexOf('repeat(') !== -1) {
      decl.value = prefixTrackValue({
        value: decl.value
      });
    }

    return _Declaration.prototype.set.call(this, decl, prefix);
  };

  return GridRowsColumns;
}(Declaration);

_defineProperty(GridRowsColumns, "names", ['grid-template-rows', 'grid-template-columns', 'grid-rows', 'grid-columns']);

module.exports = GridRowsColumns;