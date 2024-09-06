<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class SessionService extends Model
{
    protected $fillable = ['repository_id', 'workflow_id', 'name', 'url', 'repository_name'];

    protected $table = 'test_session_services';

    public function session(): BelongsTo
    {
        return $this->belongsTo(Session::class);
    }

    public function runs(): HasMany
    {
        return $this->hasMany(SessionServiceRun::class, 'service_id');
    }

    public function suites(): HasMany
    {
        return $this->hasMany(SessionServiceSuite::class, 'service_id');
    }

    public function repositoryNameWithoutOwner(): Attribute
    {
        return new Attribute(get: fn() => explode('/', $this->repository_name)[1]);
    }

    public function displayName(): Attribute
    {
        return new Attribute(get: fn() => $this->repositoryNameWithoutOwner . '/' . $this->name);
    }
}
