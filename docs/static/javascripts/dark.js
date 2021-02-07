// adapted from https://radu-matei.com/blog/dark-mode/
window.onload = function(){
  var toggle = document.getElementById("dark-mode-toggle");
  var darkTheme = document.getElementById("dark-mode-theme");

  // the default theme is light
  var savedTheme = localStorage.getItem("dark-mode-storage") || "dark";
  setTheme(savedTheme);

  toggle.addEventListener("click", () => {
    if (toggle.className === "fa fa-moon-o") {
      setTheme("dark");
    } else if (toggle.className === "fa fa-sun-o") {
      setTheme("light");
    }
  });

  function setTheme(mode) {
    localStorage.setItem("dark-mode-storage", mode);
  
    if (mode === "dark") {
      darkTheme.disabled = false;
      toggle.className = "fa fa-sun-o";
      toggle.title = "Enable Light Mode";
    } else if (mode === "light") {
      darkTheme.disabled = true;
      toggle.className = "fa fa-moon-o";
      toggle.title = "Enable Dark Mode";
    }
  }
}