(function($) {

    'use strict';

    var Search = {
        init: function() {
            $(document).ready(function() {
               $(document).on('keypress', '.td-search-input', function(e) {
                    if (e.keyCode !== 13) {
                        return
                    }

                    var query = $(this).val();
                    var searchPage = "{{ "search/" | absURL }}?q=" + query;
                    document.location = searchPage;

                    return false;
                });

            });
        },
    };

    Search.init();


}(jQuery));