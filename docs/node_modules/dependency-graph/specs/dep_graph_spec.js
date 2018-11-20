var DepGraph = require('../lib/dep_graph').DepGraph;

describe('DepGraph', function () {

  it('should be able to add/remove nodes', function () {
    var graph = new DepGraph();

    graph.addNode('Foo');
    graph.addNode('Bar');

    expect(graph.hasNode('Foo')).toBe(true);
    expect(graph.hasNode('Bar')).toBe(true);
    expect(graph.hasNode('NotThere')).toBe(false);

    graph.removeNode('Bar');

    expect(graph.hasNode('Bar')).toBe(false);
  });

  it('should calculate its size', function () {
    var graph = new DepGraph();

    expect(graph.size()).toEqual(0);

    graph.addNode('Foo');
    graph.addNode('Bar');

    expect(graph.size()).toEqual(2);

    graph.removeNode('Bar');

    expect(graph.size()).toEqual(1);
  });

  it('should treat the node data parameter as optional and use the node name as data if node data was not given', function () {
    var graph = new DepGraph();

    graph.addNode('Foo');

    expect(graph.getNodeData('Foo')).toBe('Foo');
  });

  it('should be able to associate a node name with data on node add', function () {
    var graph = new DepGraph();

    graph.addNode('Foo', 'data');

    expect(graph.getNodeData('Foo')).toBe('data');
  });

  it('should be able to add undefined as node data', function () {
    var graph = new DepGraph();

    graph.addNode('Foo', undefined);

    expect(graph.getNodeData('Foo')).toBe(undefined);
  });

  it('should return true when using hasNode with a node which has falsy data', function () {
    var graph = new DepGraph();

    var falsyData = ['', 0, null, undefined, false];
    graph.addNode('Foo');

    falsyData.forEach(function(data) {
      graph.setNodeData('Foo', data);

      expect(graph.hasNode('Foo')).toBe(true);

      // Just an extra check to make sure that the saved data is correct
      expect(graph.getNodeData('Foo')).toBe(data);
    });
  });

  it('should be able to set data after a node was added', function () {
    var graph = new DepGraph();

    graph.addNode('Foo', 'data');
    graph.setNodeData('Foo', 'data2');

    expect(graph.getNodeData('Foo')).toBe('data2');
  });

  it('should throw an error if we try to set data for a non-existing node', function () {
    var graph = new DepGraph();

    expect(function () {
      graph.setNodeData('Foo', 'data');
    }).toThrow(new Error('Node does not exist: Foo'));
  });

  it('should throw an error if the node does not exists and we try to get data', function () {
    var graph = new DepGraph();

    expect(function () {
      graph.getNodeData('Foo');
    }).toThrow(new Error('Node does not exist: Foo'));
  });

  it('should do nothing if creating a node that already exists', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');

    graph.addDependency('a','b');

    graph.addNode('a');

    expect(graph.dependenciesOf('a')).toEqual(['b']);
  });

  it('should do nothing if removing a node that does not exist', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    expect(graph.hasNode('a')).toBe(true);

    graph.removeNode('a');
    expect(graph.hasNode('Foo')).toBe(false);

    graph.removeNode('a');
    expect(graph.hasNode('Foo')).toBe(false);
  });

  it('should be able to add dependencies between nodes', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');

    graph.addDependency('a','b');
    graph.addDependency('a','c');

    expect(graph.dependenciesOf('a')).toEqual(['b', 'c']);
  });

  it('should throw an error if a node does not exist and a dependency is added', function () {
    var graph = new DepGraph();

    graph.addNode('a');

    expect(function () {
      graph.addDependency('a','b');
    }).toThrow(new Error('Node does not exist: b'));
  });

  it('should detect cycles', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addNode('d');

    graph.addDependency('a', 'b');
    graph.addDependency('b', 'c');
    graph.addDependency('c', 'a');
    graph.addDependency('d', 'a');

    expect(function () {
      graph.dependenciesOf('b');
    }).toThrow(new Error('Dependency Cycle Found: b -> c -> a -> b'));
  });

  it('should allow cycles when configured', function () {
    var graph = new DepGraph({ circular: true });

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addNode('d');

    graph.addDependency('a', 'b');
    graph.addDependency('b', 'c');
    graph.addDependency('c', 'a');
    graph.addDependency('d', 'a');

    expect(graph.dependenciesOf('b')).toEqual(['a', 'c']);
    expect(graph.overallOrder()).toEqual(['c', 'b', 'a', 'd']);
  });

  it('should detect cycles in overall order', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addNode('d');

    graph.addDependency('a', 'b');
    graph.addDependency('b', 'c');
    graph.addDependency('c', 'a');
    graph.addDependency('d', 'a');

    expect(function () {
      graph.overallOrder();
    }).toThrow(new Error('Dependency Cycle Found: a -> b -> c -> a'));
  });

  it('should detect cycles in overall order when all nodes have dependants (incoming edges)', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');

    graph.addDependency('a', 'b');
    graph.addDependency('b', 'c');
    graph.addDependency('c', 'a');

    expect(function () {
      graph.overallOrder();
    }).toThrow(new Error('Dependency Cycle Found: a -> b -> c -> a'));
  });

  it('should detect cycles in overall order when there are several ' +
     'disconnected subgraphs (with one that does not have a cycle', function () {
    var graph = new DepGraph();

    graph.addNode('a_1');
    graph.addNode('a_2');
    graph.addNode('b_1');
    graph.addNode('b_2');
    graph.addNode('b_3');

    graph.addDependency('a_1', 'a_2');
    graph.addDependency('b_1', 'b_2');
    graph.addDependency('b_2', 'b_3');
    graph.addDependency('b_3', 'b_1');

    expect(function () {
      graph.overallOrder();
    }).toThrow(new Error('Dependency Cycle Found: b_1 -> b_2 -> b_3 -> b_1'));
  });

  it('should retrieve dependencies and dependants in the correct order', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addNode('d');

    graph.addDependency('a', 'd');
    graph.addDependency('a', 'b');
    graph.addDependency('b', 'c');
    graph.addDependency('d', 'b');

    expect(graph.dependenciesOf('a')).toEqual(['c', 'b', 'd']);
    expect(graph.dependenciesOf('b')).toEqual(['c']);
    expect(graph.dependenciesOf('c')).toEqual([]);
    expect(graph.dependenciesOf('d')).toEqual(['c', 'b']);

    expect(graph.dependantsOf('a')).toEqual([]);
    expect(graph.dependantsOf('b')).toEqual(['a','d']);
    expect(graph.dependantsOf('c')).toEqual(['a','d','b']);
    expect(graph.dependantsOf('d')).toEqual(['a']);
  });

  it('should be able to resolve the overall order of things', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addNode('d');
    graph.addNode('e');

    graph.addDependency('a', 'b');
    graph.addDependency('a', 'c');
    graph.addDependency('b', 'c');
    graph.addDependency('c', 'd');

    expect(graph.overallOrder()).toEqual(['d', 'c', 'b', 'a', 'e']);
  });

  it('should be able to only retrieve the "leaves" in the overall order', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addNode('d');
    graph.addNode('e');

    graph.addDependency('a', 'b');
    graph.addDependency('a', 'c');
    graph.addDependency('b', 'c');
    graph.addDependency('c', 'd');

    expect(graph.overallOrder(true)).toEqual(['d', 'e']);
  });

  it('should be able to give the overall order for a graph with several disconnected subgraphs', function () {
    var graph = new DepGraph();

    graph.addNode('a_1');
    graph.addNode('a_2');
    graph.addNode('b_1');
    graph.addNode('b_2');
    graph.addNode('b_3');

    graph.addDependency('a_1', 'a_2');
    graph.addDependency('b_1', 'b_2');
    graph.addDependency('b_2', 'b_3');

    expect(graph.overallOrder()).toEqual(['a_2', 'a_1', 'b_3', 'b_2', 'b_1']);
  });

  it('should give an empty overall order for an empty graph', function () {
    var graph = new DepGraph();

    expect(graph.overallOrder()).toEqual([]);
  });

  it('should still work after nodes are removed', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addDependency('a', 'b');
    graph.addDependency('b', 'c');

    expect(graph.dependenciesOf('a')).toEqual(['c', 'b']);

    graph.removeNode('c');

    expect(graph.dependenciesOf('a')).toEqual(['b']);
  });

  it('should clone an empty graph', function () {
    var graph = new DepGraph();
    expect(graph.size()).toEqual(0);
    var cloned = graph.clone();
    expect(cloned.size()).toEqual(0);

    expect(graph === cloned).toBe(false);
  });

  it('should clone a non-empty graph', function () {
    var graph = new DepGraph();

    graph.addNode('a');
    graph.addNode('b');
    graph.addNode('c');
    graph.addDependency('a', 'b');
    graph.addDependency('b', 'c');

    var cloned = graph.clone();

    expect(graph === cloned).toBe(false);
    expect(cloned.hasNode('a')).toBe(true);
    expect(cloned.hasNode('b')).toBe(true);
    expect(cloned.hasNode('c')).toBe(true);
    expect(cloned.dependenciesOf('a')).toEqual(['c', 'b']);
    expect(cloned.dependantsOf('c')).toEqual(['a', 'b']);

    // Changes to the original graph shouldn't affect the clone
    graph.removeNode('c');
    expect(graph.dependenciesOf('a')).toEqual(['b']);
    expect(cloned.dependenciesOf('a')).toEqual(['c', 'b']);

    graph.addNode('d');
    graph.addDependency('b', 'd');
    expect(graph.dependenciesOf('a')).toEqual(['d', 'b']);
    expect(cloned.dependenciesOf('a')).toEqual(['c', 'b']);
  });

  it('should only be a shallow clone', function () {
    var graph = new DepGraph();

    var data = {a: 42};
    graph.addNode('a', data);

    var cloned = graph.clone();
    expect(graph === cloned).toBe(false);
    expect(graph.getNodeData('a') === cloned.getNodeData('a')).toBe(true);

    graph.getNodeData('a').a = 43;
    expect(cloned.getNodeData('a').a).toEqual(43);

    cloned.setNodeData('a', {a: 42});
    expect(cloned.getNodeData('a').a).toEqual(42);
    expect(graph.getNodeData('a') === cloned.getNodeData('a')).toBe(false);
  });
});
