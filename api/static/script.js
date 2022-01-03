document.addEventListener("DOMContentLoaded", function () {
    var examples = document.querySelectorAll("div.example-block");

    for (let i = 0; i < examples.length; i++) {
        examples[i].addEventListener("click", function () {
            this.classList.toggle("active");
            var content = this.nextElementSibling;
            if (content.style.display === "block") {
                content.style.display = "none";
            } else {
                content.style.display = "block";
            }
        });
    }
});
