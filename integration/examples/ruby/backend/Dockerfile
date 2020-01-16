FROM ruby:2.7

WORKDIR /app
COPY . .
RUN bundle install

CMD ["bundle","exec","puma"]
