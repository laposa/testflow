<?php

use App\Http\Controllers\AuthController;
use Illuminate\Support\Facades\Route;


Route::post('/auth/redirect', [AuthController::class, 'redirect']);
Route::get('/auth/callback', [AuthController::class, 'callback']);
Route::get('/auth/logout', [AuthController::class, 'logout']);

Route::get('/login', [AuthController::class, 'login'])->name('login');

Route::group(['middleware' => 'auth'], function () {
    Route::get('/', function () {
        return view('dashboard.index');
    })->name('dashboard');

    /**
     * Github App Routes
     */
    Route::get('/github/callback', [\App\Http\Controllers\GithubAppController::class, 'callback']);

    Route::get('/sessions', [\App\Http\Controllers\SessionsController::class, 'index'])->name('sessions.index');
    Route::get('/sessions/create', [\App\Http\Controllers\SessionsController::class, 'create'])->name('sessions.create');
    Route::get('/sessions/{session}', [\App\Http\Controllers\SessionsController::class, 'show'])->name('sessions.show');
    Route::post('/sessions', [\App\Http\Controllers\SessionsController::class, 'store'])->name('sessions.store');

    Route::get('/sessions/{session}/test/{test}/run/{run}', [\App\Http\Controllers\RunsController::class, 'show'])->name('session.test.runs.show');

    Route::get('/tests', [\App\Http\Controllers\TestsController::class, 'index']);
    Route::get('/tests/{test}', [\App\Http\Controllers\TestsController::class, 'show']);
    Route::get('/runs', [\App\Http\Controllers\RunsController::class, 'index']);
    Route::get('/maintenance', [\App\Http\Controllers\MaintenanceController::class, 'index']);


});
