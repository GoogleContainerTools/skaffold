FROM ruby:2.7

WORKDIR /app
ADD Gemfile* ./
RUN bundle install

ADD . ./

CMD ["bundle","exec","puma"]
