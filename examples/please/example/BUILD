subinclude("@pleasings//docker:docker")



python_library(
    name = 'example',
    srcs = [ 'example.py' ],
    zip_safe = True,
)

docker_image(
    name = 'base',
    dockerfile = 'Dockerfile-base',
)

docker_image(
    name = 'image',
    srcs = [':example'],
    base_image = ':base',
    run_args = '-p 8000:8000',
    #visibilty = ['PUBLIC']
)