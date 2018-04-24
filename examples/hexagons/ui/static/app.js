"use strict";

angular.module('quizz', ['ngRoute'])

.controller('QuizzCtrl', function($scope, $routeParams, $http, $route) {
  $scope.params = $routeParams;

  $http.get('/api/hexagons').then(function(reponse) {
    var hexagons = reponse.data;
    var count = hexagons.length
    var randomGuess = Math.floor(Math.random()*count);
    var randomChoice = randomGuess
    while (randomChoice == randomGuess) {
      randomChoice = Math.floor(Math.random()*count);
    }

    $scope.guess = hexagons[randomGuess];
    $scope.flavor = Math.random() >= 0.5 ? 1 : 2
    $scope.playAgain = false;
    $scope.play = $route.reload;

    var win = function() {
      if ($scope.playAgain) return;
      $scope.correct = true;
      $scope.wrong = false;
      $scope.playAgain = true;
    };
    var lose = function() {
      if ($scope.playAgain) return;
      $scope.correct = false;
      $scope.wrong = true;
      $scope.playAgain = true;
    };

    if (Math.random() >= 0.5) {
      $scope.candidate1 = hexagons[randomGuess];
      $scope.candidate2 = hexagons[randomChoice];
      $scope.playLeft = win;
      $scope.playRight = lose;
    } else {
      $scope.candidate1 = hexagons[randomChoice];
      $scope.candidate2 = hexagons[randomGuess];
      $scope.playLeft = lose;
      $scope.playRight = win;
    }
  });
})

.config(function($routeProvider, $locationProvider, $httpProvider) {
  $routeProvider
    .when('/', { templateUrl: '/templates/quizz.html', controller: 'QuizzCtrl' })

  $locationProvider.html5Mode(true);
});
