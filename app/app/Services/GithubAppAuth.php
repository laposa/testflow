<?php

namespace App\Services;

use App\Data\GithubAppAuthData;
use Illuminate\Support\Facades\Cache;

use Carbon\Carbon;
use Illuminate\Validation\UnauthorizedException;

class GithubAppAuth
{
    static public function get(): ?GithubAppAuthData
    {
        $data = Cache::get('installation');

        if (!$data) {
            throw new UnauthorizedException('Installation not found.');
        }

        return  $data ? GithubAppAuthData::from($data): null;
    }

    static public function exists(): bool
    {
        return Cache::has('installation');
    }

    static public function set()
    {
        $data = GithubClient::getAccessToken(env("GITHUB_APP_INSTALATION_ID"), GithubClient::generateJWTWebToken());
        self::update($data);
    }

    static public function update(array $data)
    {
        $cacheDuration = now()->diffInSeconds(Carbon::parse($data['expires_at']));
        Cache::put(
            key: 'installation',
            value: GithubAppAuthData::from([
                'access_token' => $data['token'],
                'expires_at' => $data['expires_at'],
                'repository_selection' => $data['repository_selection'],
            ])->toArray(),
            ttl: $cacheDuration
        );
    }

    static public function connect(): bool
    {
        if (self::exists()) {
            $installation = self::get();
            if (now()->gte($installation->expires_at)) {
                self::set();
            }
        } else {
            self::set();
        }

        return self::exists();

    }
}
