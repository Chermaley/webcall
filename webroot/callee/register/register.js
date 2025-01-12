// WebCall Copyright 2022 timur.mobi. All rights reserved.
'use strict';
const registerButton = document.querySelector('button#register');
const statusLine = document.getElementById('status');
const idLine = document.getElementById('id');
const form = document.querySelector('form#password');
const formPw = document.querySelector('input#pw');
var myCalleeID = "";
var calleeLink = "";

window.onload = function() {
	showStatus("<br><br>please wait...<br><br><br><br><br>",-1);
	makeNewId(); // -> isAvailAction()
}

function makeNewId() {
	let api = apiPath+"/newid";
	if(typeof Android !== "undefined" && Android !== null) {
		if(typeof Android.getVersionName !== "undefined" && Android.getVersionName !== null) {
			api = api + "?ver="+Android.getVersionName();
		}
		if(typeof Android.webviewVersion !== "undefined" && Android.webviewVersion !== null) {
			api = api + "_" + Android.webviewVersion();
		}
	} else {
		//api = api + "&ver="+clientVersion;
	}
	if(!gentle) console.log('request newid api',api);
	ajaxFetch(new XMLHttpRequest(), "GET", api, function(xhr) {
		if(!gentle) console.log('xhr.responseText',xhr.responseText);
		if(xhr.responseText=="") {
			showStatus("Registration of new callee ID's is not possible at this time. Please try again later. Thank you.<br><br><a href='..'>Back</a>",-1);
		} else if(xhr.responseText.length!=11) {
			showStatus("Error: "+xhr.responseText+"<br><br><a href='..'>Back</a>",-1);
		} else {
			// myCalleeID = xhr.responseText;
			isAvailAction();
		}
	}, errorAction);
}

function isAvailAction() {
	showStatus("<br>Создайте id звонка.",-1);
	// show form and clear pw input field
	form.style.display = "block";
	setTimeout(function() {
		console.log('formPw.focus');
		formPw.focus();
	},400);
	// pw confirmation will take place in submitForm()
}

function errorAction(errString,err) {
	console.log('xhr error',errString);
	showStatus('xhr error '+errString,-1);
}

var xhrTimeout = 5000;
function ajaxFetch(xhr, type, apiPath, processData, errorFkt, postData) {
	xhr.onreadystatechange = function() {
		if(xhr.readyState == 4 && (xhr.status==200 || xhr.status==0)) {
			processData(xhr);
		} else if(xhr.readyState==4) {
			errorFkt("fetch error",xhr.status);
		}
	}
	xhr.timeout = xhrTimeout;
	xhr.ontimeout = function () {
		errorFkt("timeout",0);
	}
	xhr.onerror= function(e) {
		errorFkt("fetching",xhr.status);
	};
	// cross-browser compatible approach to bypassing the cache
	if(apiPath.indexOf("?")>=0) {
		apiPath += "&_="+new Date().getTime();
	} else {
		apiPath += "?_="+new Date().getTime();
	}
	if(!gentle) console.log('xhr send',apiPath);
	xhr.open(type, apiPath, true);
	xhr.setRequestHeader("Content-type", "text/plain; charset=utf-8");
	if(postData) {
		xhr.send(postData);
	} else {
		xhr.send();
	}
}

function showStatus(msg,timeoutMs) {
	let sleepMs = 2500;
	if(typeof timeoutMs!=="undefined") {
		sleepMs = timeoutMs;
	}
	statusLine.style.display = "none";
	statusLine.style.opacity = 0;
	statusLine.innerHTML = msg;
	statusLine.style.opacity = 1;
	statusLine.style.display = "block";
	if(msg!="" && sleepMs>=0) {
		setTimeout(function(oldMsg) {
			if(statusLine.innerHTML==oldMsg) {
				statusLine.style.opacity = 0;
			}
		},sleepMs,msg);
	}
}

function submitForm(theForm) {
	//if(!gentle) cconsole.log("submitForm",theForm);
	// var valuePw = document.getElementById("pw").value;
	// if(!gentle) console.log('submitForm valuePw.length',valuePw.length);
	// if(valuePw.length < 6) {
	// 	showStatus("Password must have six or more characters",-1);
	// 	return;
	// }
	myCalleeID = document.getElementById("username").value;

	form.style.display = "none";
	showStatus("Register new ID...")
	setTimeout(function() {
		// register new ID
		let api = apiPath+"/register/"+myCalleeID;
		if(typeof Android !== "undefined" && Android !== null) {
			if(typeof Android.getVersionName !== "undefined" && Android.getVersionName !== null) {
				api = api + "?ver="+Android.getVersionName();
			}
			if(typeof Android.webviewVersion !== "undefined" && Android.webviewVersion !== null) {
				api = api + "_" + Android.webviewVersion();
			}
		} else {
			//api = api + "&ver="+clientVersion;
		}
		if(!gentle) console.log('register via api='+api);
		ajaxFetch(new XMLHttpRequest(), "POST", api, function(xhr) {
			if(xhr.responseText=="OK") {
				// ID is registered; offer the link
				calleeLink = window.location.href;
				// calleeLink may have ?i=906735 attached: cut it off
				let idxArg = calleeLink.indexOf("?");
				if(idxArg>=0) calleeLink = calleeLink.substring(0,idxArg);
				//if(!gentle) console.log('calleeLink1='+calleeLink);
				calleeLink = calleeLink.replace("register/","");
				//if(!gentle) console.log('calleeLink2='+calleeLink+" myCalleeID="+myCalleeID);
				calleeLink += myCalleeID;
				if(!gentle) console.log('calleeLink='+calleeLink);
				window.location.href = calleeLink;
			} else {
				console.log('response:',xhr.responseText);
				showStatus("Sorry, it is not possible to register your ID right now. Please try again a little later.",-1);
			}
		}, errorAction, "pw="+'valuePw');
	},2000);
}

function exelink(url) {
	console.log("exelink parent", window.location, window.parent.location);
	if(window.location !== window.parent.location) {
		// running inside an iframe -> open in a new tab
		//console.log("exelink open",calleeLink);
		window.open(calleeLink, '_blank');
	} else {
		// not running inside an iframe -> continue in the same tab
		//console.log("exelink replace",calleeLink);
		window.location.replace(calleeLink+"?auto=1");
	}
}

function clearForm() {
	document.getElementById("pw").value = "";
	formPw.focus();
}

function showPw() {
	if(formPw.type=="password") {
		formPw.type="text";
	} else {
		formPw.type="password";
	}
}

