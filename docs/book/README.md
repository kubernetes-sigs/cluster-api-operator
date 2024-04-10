# Preview book changes locally

It is easy to preview your local changes to the book before submitting a PR:

1. Build the local copy of the book from the `docs/book` path:

    ```shell
    make build
    ```

1. To preview the book contents run:

    ```shell
    make serve
    ```

This should serve the book at [localhost:3000](http://localhost:3000/). You can keep running `make serve` and continue making doc changes. mdBook will detect your changes, render them and refresh your browser page automatically.

1. Clean mdBook auto-generated content from `docs/book/book` path once you have finished local preview:

    ```shell
    make clean
    ```