'use strict'
const path = require('path')
const DepGraph = require('dependency-graph').DepGraph

const graph = new DepGraph()

exports.add = message => {
  message.parent = path.resolve(message.parent)
  message.file = path.resolve(message.file)

  graph.addNode(message.parent)
  graph.addNode(message.file)
  graph.addDependency(message.parent, message.file)
  return message
}

exports.dependantsOf = node => {
  node = path.resolve(node)

  if (graph.hasNode(node)) return graph.dependantsOf(node)
  return []
}
