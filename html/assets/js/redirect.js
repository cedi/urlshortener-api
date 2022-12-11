function redirect(redirectAfter, redirectTo) {
    setTimeout(function () {
        window.location.replace(redirectTo);
    }, redirectAfter * 1000);
}

function decrement(redirectAfter) {
    if (!redirectAfter) {
        return;
    }

    let counterElem = document.querySelector('.redirectAfter');
    let counter = 1 + redirectAfter;

    function decr() {
        counter--;
        counterElem.innerHTML = counter;
    }

    (function (val) {
        while (val--) { setTimeout(decr, 1000 * val); }
    })(counter);
}
