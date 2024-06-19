<?php

namespace App\Actions\Session;


use App\Models\Installation;
use Illuminate\Support\Facades\Validator;

class CreateSession
{
    public function handle(Installation $installation, array $data)
    {
        $validated = Validator::make($data, [
            'name' => ['required', 'string'],
            'data' => ['required', 'array', 'min:1'],
            'data.*' => ['required', 'json'],
        ], [
            'data' => 'Please select at least one test suite.',

        ])->validate();

        $session = $installation->sessions()->create([
            'name' => $validated['name'],
            'issuer_id' => auth()->id(),
            'data' => collect($validated["data"])->map(fn($item) => json_decode($item, true)),
        ]);
        $session->save();

        return $session;
    }
}
