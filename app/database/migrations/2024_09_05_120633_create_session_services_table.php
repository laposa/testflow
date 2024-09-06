<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('test_session_services', function (Blueprint $table) {
            $table->id();
            $table
                ->foreignId('session_id')
                ->constrained('test_sessions')
                ->onDelete('cascade')
                ->onUpdate('cascade');

            $table->unsignedInteger('repository_id');
            $table->unsignedInteger('workflow_id');
            $table->string('name');
            $table->string('url');
            $table->string('repository_name');

            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('session_services');
    }
};
