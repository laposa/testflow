<?php

namespace App\Http\Controllers;

use App\Actions\Github\InstallGithubApp;

class GithubAppController extends Controller
{
    public function callback(InstallGithubApp $installGithubApp)
    {

        $installGithubApp->handle( request('installation_id'), request('setup_action'));

        return redirect()->route('dashboard');
    }

}
