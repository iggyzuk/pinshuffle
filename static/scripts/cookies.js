function getCookie(cname) {
    const name = cname + "=";
    for (const part of document.cookie.split(';')) {
        const cookie = part.trimStart();
        if (cookie.startsWith(name)) {
            return cookie.substring(name.length);
        }
    }
    return "";
}

function setCookie(cname, cvalue, exdays) {
    const d = new Date();
    d.setTime(d.getTime() + (exdays * 24 * 60 * 60 * 1000));
    document.cookie = cname + "=" + cvalue + ";expires=" + d.toUTCString() + ";path=/;SameSite=Lax";
}
