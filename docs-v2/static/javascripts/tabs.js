function initTabs() {
    $('.tab-content').children('.tab-pane').each(function(idx, item) {
      var navTabs = $(this).closest('.code-tabs').children('.nav-tabs'),
          title = $(this).attr('title');
      navTabs.append('<li class="nav-tab"><a href="#" class="nav-tab">'+title+'</a></li');
    });
   
    $('.code-tabs ul.nav-tabs').each(function() {
      $(this).find("li:first").addClass('active');
    })
  
    $('.code-tabs .tab-content').each(function() {
      $(this).find("div:first").addClass('active');
    });
  
    $('.nav-tabs a').click(function(e){
      e.preventDefault();
      var tab = $(this).parent(),
          tabIndex = tab.index(),
          tabPanel = $(this).closest('.code-tabs'),
          tabPane = tabPanel.find('.tab-content:first').children('.tab-pane').eq(tabIndex);
      tab.siblings().removeClass('active');
      tabPane.siblings().removeClass('active');
      tab.addClass('active');
      tabPane.addClass('active');
    });
  }
