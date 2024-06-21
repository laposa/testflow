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
        Schema::create('test_session_items', function (Blueprint $table) {
            $table->id();
            $table->foreignId('session_id')
                ->constrained('test_sessions')
                ->onDelete('cascade')
                ->onUpdate('cascade');
            $table->unsignedInteger('repository_id');
            $table->unsignedInteger('workflow_id');
            $table->string('repository_name');
            $table->string('service_name');
            $table->string('suite_name');
            $table->string('test_name');
            $table->string('service_url');
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('test_session_items');
    }
};
