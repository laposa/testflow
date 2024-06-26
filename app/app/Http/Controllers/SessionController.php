<?php

namespace App\Http\Controllers;

use App\Actions\Session\CreateSession;
use App\Models\Installation;
use App\Models\Session;
use Illuminate\Http\Request;

class SessionController extends Controller
{
    public function index() {
        return view('sessions.index', [
            'sessions' => Session::all(),
        ]);
    }

    public function show(Session $session) {
        return view('sessions.show', [
            'session' => $session
        ]);
    }

    public function create() {
        return view('sessions.create');
    }

    public function store(Request $request, CreateSession $createSession) {
        $session = $createSession->handle(Installation::first(), $request->all());

        return redirect()->route('sessions.show', $session);
    }

}
