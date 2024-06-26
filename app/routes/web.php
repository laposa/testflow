<?php

use App\Http\Controllers\AuthController;
use Illuminate\Support\Facades\Route;


Route::post('/auth/redirect', [AuthController::class, 'redirect']);
Route::get('/auth/callback', [AuthController::class, 'callback']);
Route::get('/auth/logout', [AuthController::class, 'logout']);

Route::get('/login', [AuthController::class, 'login'])->name('login');

Route::group(['middleware' => 'auth'], function () {

    // landing page showing who is logged in to the portal - authorised with GitHub
    Route::get('/', function () {
        return view('dashboard.index');
    });

    // Github App Routes
    Route::get('/github/callback', [\App\Http\Controllers\GithubAppController::class, 'callback']);

    // list of previous sessions, allow create new session
    Route::get('/sessions', [\App\Http\Controllers\SessionController::class, 'index'])->name('sessions.index');
    // show list of all available tests loaded from GitHub - automated and manual and allow to select any of them
    Route::get('/sessions/create', [\App\Http\Controllers\SessionController::class, 'create'])->name('sessions.create');
    Route::post('/sessions', [\App\Http\Controllers\SessionController::class, 'store'])->name('sessions.store');
    Route::post('/sessions/create', [\App\Http\Controllers\SessionController::class, 'store'])->name('sessions.store');
    // show selected tests and their previous runs history and allow to execute another run
    Route::get('/sessions/{session}', [\App\Http\Controllers\SessionController::class, 'show'])->name('sessions.show');
    /**
     * for manual, checkboxes with option to add comments
     * for automated, show button to execute
     */
     Route::get('/sessions/{session}/run/{run}', [\App\Http\Controllers\SessionRunController::class, 'show'])->name('session.runs.show');
});
