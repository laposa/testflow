<?php

use App\Http\Controllers\AuthController;
use Illuminate\Support\Facades\Route;


Route::post('/auth/redirect', [AuthController::class, 'redirect']);
Route::post('/auth/callback', [AuthController::class, 'callback']);

Route::get('/login', [AuthController::class, 'login'])->name('login');

Route::group(['middleware' => 'auth'], function () {
    Route::get('/', function () {
        return view('welcome');
    });
});
